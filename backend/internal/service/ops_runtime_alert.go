package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const (
	OpsRuntimeAlertTypeOpenAIAccountCircuitOpen    = "openai_account_circuit_open"
	OpsRuntimeAlertTypeOpenAISelectionEmpty        = "openai_selection_empty"
	opsRuntimeAlertDefaultDedupWindow              = 2 * time.Minute
	opsRuntimeAlertContextTimeout                  = 5 * time.Second
	opsRuntimeAlertCircuitSourceType               = "ops_runtime_alert"
	opsRuntimeAlertOpenAISelectionEmptyDescription = "OpenAI scheduler returned no usable account after filters"
)

type OpsRuntimeAlertService struct {
	opsRepo      OpsRepository
	opsService   *OpsService
	emailService *EmailService

	mu           sync.Mutex
	lastSent     map[string]time.Time
	emailLimiter *slidingWindowLimiter
}

type OpsRuntimeAlertInput struct {
	Type        string
	Severity    string
	Title       string
	Description string
	Dimensions  map[string]any
	DedupKey    string
	DedupWindow time.Duration
}

func NewOpsRuntimeAlertService(opsRepo OpsRepository, opsService *OpsService, emailService *EmailService) *OpsRuntimeAlertService {
	return &OpsRuntimeAlertService{
		opsRepo:      opsRepo,
		opsService:   opsService,
		emailService: emailService,
		lastSent:     map[string]time.Time{},
		emailLimiter: newSlidingWindowLimiter(0, time.Hour),
	}
}

func (s *OpsRuntimeAlertService) Emit(ctx context.Context, input OpsRuntimeAlertInput) {
	if s == nil || s.opsRepo == nil || s.opsService == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	alertType := strings.TrimSpace(input.Type)
	if alertType == "" {
		return
	}
	if !s.opsService.IsMonitoringEnabled(ctx) {
		return
	}

	now := time.Now().UTC()
	dedupWindow := input.DedupWindow
	if dedupWindow <= 0 {
		dedupWindow = opsRuntimeAlertDefaultDedupWindow
	}
	dedupKey := strings.TrimSpace(input.DedupKey)
	if dedupKey == "" {
		dedupKey = alertType
	}
	if !s.allow(dedupKey, now, dedupWindow) {
		return
	}

	event := &OpsAlertEvent{
		Severity:    normalizeOpsRuntimeAlertSeverity(input.Severity),
		Status:      OpsAlertStatusFiring,
		Title:       truncateString(strings.TrimSpace(input.Title), 200),
		Description: truncateString(strings.TrimSpace(input.Description), 2000),
		Dimensions:  sanitizeOpsRuntimeAlertDimensions(input.Dimensions),
		FiredAt:     now,
	}
	if event.Title == "" {
		event.Title = alertType
	}
	if event.Description == "" {
		event.Description = event.Title
	}
	if event.Dimensions == nil {
		event.Dimensions = map[string]any{}
	}
	event.Dimensions["alert_type"] = alertType

	alertCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), opsRuntimeAlertContextTimeout)
	defer cancel()

	created, err := s.opsRepo.CreateAlertEvent(alertCtx, event)
	if err != nil {
		slog.Warn("ops_runtime_alert_create_failed", "alert_type", alertType, "dedup_key", dedupKey, "error", err)
		return
	}
	if created == nil {
		created = event
	}
	if created.ID > 0 {
		s.maybeSendEmail(alertCtx, created)
	}
}

func (s *OpsRuntimeAlertService) allow(key string, now time.Time, window time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastSent == nil {
		s.lastSent = map[string]time.Time{}
	}
	if last, ok := s.lastSent[key]; ok && now.Sub(last) < window {
		return false
	}
	s.lastSent[key] = now
	return true
}

func (s *OpsRuntimeAlertService) maybeSendEmail(ctx context.Context, event *OpsAlertEvent) {
	if s == nil || s.emailService == nil || s.opsService == nil || event == nil || event.EmailSent {
		return
	}
	emailCfg, err := s.opsService.GetEmailNotificationConfig(ctx)
	if err != nil || emailCfg == nil || !emailCfg.Alert.Enabled || len(emailCfg.Alert.Recipients) == 0 {
		return
	}
	if !shouldSendOpsAlertEmailByMinSeverity(strings.TrimSpace(emailCfg.Alert.MinSeverity), strings.TrimSpace(event.Severity)) {
		return
	}
	runtimeCfg, _ := s.opsService.GetOpsAlertRuntimeSettings(ctx)
	if runtimeCfg != nil && runtimeCfg.Silencing.Enabled {
		rule := &OpsAlertRule{Name: event.Title, Severity: event.Severity, MetricType: "runtime", Operator: ">=", Threshold: 1}
		if isOpsAlertSilenced(time.Now().UTC(), rule, event, runtimeCfg.Silencing) {
			return
		}
	}

	s.emailLimiter.SetLimit(emailCfg.Alert.RateLimitPerHour)
	rule := &OpsAlertRule{
		Name:        event.Title,
		Description: event.Description,
		Severity:    event.Severity,
		MetricType:  "runtime",
		Operator:    ">=",
		Threshold:   1,
	}
	subject := fmt.Sprintf("[Ops Alert][%s] %s", strings.TrimSpace(event.Severity), strings.TrimSpace(event.Title))
	body := buildOpsAlertEmailBody(rule, event)

	anySent := false
	for _, to := range emailCfg.Alert.Recipients {
		addr := strings.TrimSpace(to)
		if addr == "" || !s.emailLimiter.Allow(time.Now().UTC()) {
			continue
		}
		if s.emailService.notificationEmailService != nil {
			if err := s.emailService.notificationEmailService.Send(ctx, NotificationEmailSendInput{
				Event:          NotificationEmailEventOpsAlert,
				RecipientEmail: addr,
				RecipientName:  emailRecipientName(addr),
				SourceType:     opsRuntimeAlertCircuitSourceType,
				SourceID:       fmt.Sprintf("%d", event.ID),
				Variables:      opsAlertEmailVariables(rule, event),
			}); err == nil {
				anySent = true
				continue
			} else if !shouldFallbackNotificationEmail(err) {
				continue
			}
		}
		if err := s.emailService.SendEmail(ctx, addr, subject, body); err == nil {
			anySent = true
		}
	}
	if anySent {
		_ = s.opsRepo.UpdateAlertEventEmailSent(context.Background(), event.ID, true)
	}
}

func normalizeOpsRuntimeAlertSeverity(severity string) string {
	severity = strings.ToUpper(strings.TrimSpace(severity))
	switch severity {
	case "P0", "P1", "P2", "P3":
		return severity
	default:
		return "P1"
	}
}

func sanitizeOpsRuntimeAlertDimensions(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		name := strings.ToLower(strings.TrimSpace(key))
		if name == "" || strings.Contains(name, "key") || strings.Contains(name, "token") || strings.Contains(name, "secret") {
			continue
		}
		out[name] = sanitizeOpsRuntimeAlertDimensionValue(value)
	}
	return out
}

func sanitizeOpsRuntimeAlertDimensionValue(value any) any {
	switch v := value.(type) {
	case nil, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return v
	case string:
		return truncateString(v, 512)
	case []int64:
		return append([]int64(nil), v...)
	case []int:
		return append([]int(nil), v...)
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, truncateString(item, 256))
		}
		return out
	case map[string]int:
		out := make(map[string]int, len(v))
		for key, item := range v {
			out[truncateString(key, 128)] = item
		}
		return out
	default:
		return truncateString(fmt.Sprint(v), 512)
	}
}
