package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

const (
	groupRateChangeNoticeDefaultWindowMinutes = 30
	groupRateChangeNoticeMaxWindowMinutes     = 24 * 60
	groupRateChangeNoticeMaxUsers             = 1000
)

type AdminServiceCustom interface {
	PreviewGroupRateChangeNotification(ctx context.Context, groupID int64, input GroupRateChangeNotificationInput) (*GroupRateChangeNotificationPreview, error)
	SendGroupRateChangeNotification(ctx context.Context, groupID int64, input GroupRateChangeNotificationInput) (*GroupRateChangeNotificationSendResult, error)
	GetGroupSchedulerHistory(ctx context.Context, groupID int64, limit int) ([]SchedulerOutboxEvent, error)
}

type GroupRateChangeNotificationInput struct {
	NewRateMultiplier float64    `json:"new_rate_multiplier"`
	WindowMinutes     int        `json:"window_minutes"`
	EffectiveAt       *time.Time `json:"effective_at,omitempty"`
	Message           string     `json:"message,omitempty"`
}

type GroupRateChangeNotificationUser struct {
	UserID       int64     `json:"user_id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	RequestCount int64     `json:"request_count"`
	ActualCost   float64   `json:"actual_cost"`
	LastUsedAt   time.Time `json:"last_used_at"`
}

type GroupRateChangeNotificationPreview struct {
	GroupID           int64                             `json:"group_id"`
	GroupName         string                            `json:"group_name"`
	OldRateMultiplier float64                           `json:"old_rate_multiplier"`
	NewRateMultiplier float64                           `json:"new_rate_multiplier"`
	WindowMinutes     int                               `json:"window_minutes"`
	EffectiveAt       time.Time                         `json:"effective_at"`
	Message           string                            `json:"message,omitempty"`
	UserCount         int                               `json:"user_count"`
	SkippedCount      int                               `json:"skipped_count"`
	Users             []GroupRateChangeNotificationUser `json:"users"`
}

type GroupRateChangeNotificationSendResult struct {
	GroupID     int64     `json:"group_id"`
	UserCount   int       `json:"user_count"`
	Sent        int       `json:"sent"`
	Skipped     int       `json:"skipped"`
	Failed      int       `json:"failed"`
	LastError   string    `json:"last_error,omitempty"`
	EffectiveAt time.Time `json:"effective_at"`
}

func (s *adminServiceImpl) PreviewGroupRateChangeNotification(ctx context.Context, groupID int64, input GroupRateChangeNotificationInput) (*GroupRateChangeNotificationPreview, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeGroupRateChangeNotificationInput(input)
	if err != nil {
		return nil, err
	}
	if s.usageLogRepo == nil {
		return nil, errors.New("usage log repository is not configured")
	}

	since := time.Now().UTC().Add(-time.Duration(normalized.WindowMinutes) * time.Minute)
	recentUsers, err := listRecentGroupUsers(s.usageLogRepo, ctx, groupID, since, groupRateChangeNoticeMaxUsers)
	if err != nil {
		return nil, err
	}

	users := make([]GroupRateChangeNotificationUser, 0, len(recentUsers))
	skipped := 0
	for _, recent := range recentUsers {
		email := strings.TrimSpace(recent.Email)
		if !isDeliverableNotificationEmail(email) {
			skipped++
			continue
		}
		users = append(users, GroupRateChangeNotificationUser{
			UserID:       recent.UserID,
			Email:        email,
			Username:     strings.TrimSpace(recent.Username),
			RequestCount: recent.RequestCount,
			ActualCost:   recent.ActualCost,
			LastUsedAt:   recent.LastUsedAt,
		})
	}

	return &GroupRateChangeNotificationPreview{
		GroupID:           group.ID,
		GroupName:         group.Name,
		OldRateMultiplier: group.RateMultiplier,
		NewRateMultiplier: normalized.NewRateMultiplier,
		WindowMinutes:     normalized.WindowMinutes,
		EffectiveAt:       *normalized.EffectiveAt,
		Message:           normalized.Message,
		UserCount:         len(users),
		SkippedCount:      skipped,
		Users:             users,
	}, nil
}

func (s *adminServiceImpl) SendGroupRateChangeNotification(ctx context.Context, groupID int64, input GroupRateChangeNotificationInput) (*GroupRateChangeNotificationSendResult, error) {
	if s.notificationEmailSvc == nil {
		return nil, errors.New("notification email service is not configured")
	}
	preview, err := s.PreviewGroupRateChangeNotification(ctx, groupID, input)
	if err != nil {
		return nil, err
	}

	result := &GroupRateChangeNotificationSendResult{
		GroupID:     preview.GroupID,
		UserCount:   preview.UserCount,
		Skipped:     preview.SkippedCount,
		EffectiveAt: preview.EffectiveAt,
	}
	sourceID := fmt.Sprintf(
		"group:%d:old:%s:new:%s:effective:%s",
		preview.GroupID,
		formatNotificationRateMultiplier(preview.OldRateMultiplier),
		formatNotificationRateMultiplier(preview.NewRateMultiplier),
		preview.EffectiveAt.UTC().Format(time.RFC3339),
	)
	for _, user := range preview.Users {
		if err := s.notificationEmailSvc.Send(ctx, NotificationEmailSendInput{
			Event:          NotificationEmailEventGroupRateChangeNotice,
			RecipientEmail: user.Email,
			RecipientName:  groupRateChangeRecipientName(user),
			UserID:         user.UserID,
			SourceType:     "group_rate_change_notice",
			SourceID:       sourceID,
			ReminderKey:    strconv.FormatInt(user.UserID, 10),
			Variables: map[string]string{
				"group_name":          preview.GroupName,
				"old_rate_multiplier": formatNotificationRateMultiplier(preview.OldRateMultiplier),
				"new_rate_multiplier": formatNotificationRateMultiplier(preview.NewRateMultiplier),
				"effective_at":        formatNotificationTime(preview.EffectiveAt),
				"window_minutes":      strconv.Itoa(preview.WindowMinutes),
				"request_count":       strconv.FormatInt(user.RequestCount, 10),
				"actual_cost":         formatNotificationAmount(user.ActualCost),
				"last_used_at":        formatNotificationTime(user.LastUsedAt),
				"admin_message":       groupRateChangeAdminMessage(preview.Message),
			},
		}); err != nil {
			result.Failed++
			result.LastError = err.Error()
			slog.Warn("failed to send group rate change notification", "group_id", preview.GroupID, "user_id", user.UserID, "err", err)
			continue
		}
		result.Sent++
	}
	return result, nil
}

func normalizeGroupRateChangeNotificationInput(input GroupRateChangeNotificationInput) (GroupRateChangeNotificationInput, error) {
	if input.NewRateMultiplier <= 0 {
		return GroupRateChangeNotificationInput{}, errors.New("new_rate_multiplier must be > 0")
	}
	if input.WindowMinutes <= 0 {
		input.WindowMinutes = groupRateChangeNoticeDefaultWindowMinutes
	}
	if input.WindowMinutes > groupRateChangeNoticeMaxWindowMinutes {
		return GroupRateChangeNotificationInput{}, fmt.Errorf("window_minutes must be <= %d", groupRateChangeNoticeMaxWindowMinutes)
	}
	if input.EffectiveAt == nil || input.EffectiveAt.IsZero() {
		effectiveAt := time.Now().UTC().Add(groupRateChangeNoticeDefaultWindowMinutes * time.Minute)
		input.EffectiveAt = &effectiveAt
	} else {
		effectiveAt := input.EffectiveAt.UTC()
		input.EffectiveAt = &effectiveAt
	}
	input.Message = strings.TrimSpace(input.Message)
	return input, nil
}

func isDeliverableNotificationEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" || isReservedEmail(email) {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && strings.EqualFold(strings.TrimSpace(addr.Address), email)
}

func groupRateChangeRecipientName(user GroupRateChangeNotificationUser) string {
	if strings.TrimSpace(user.Username) != "" {
		return strings.TrimSpace(user.Username)
	}
	return strings.TrimSpace(user.Email)
}

func groupRateChangeAdminMessage(message string) string {
	if strings.TrimSpace(message) == "" {
		return "-"
	}
	return strings.TrimSpace(message)
}

func formatNotificationRateMultiplier(value float64) string {
	text := strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 4, 64), "0"), ".")
	if text == "" {
		return "0"
	}
	return text
}

func formatNotificationAmount(value float64) string {
	text := strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 6, 64), "0"), ".")
	if text == "" {
		return "0"
	}
	return text
}

func formatNotificationTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format("2006-01-02 15:04 UTC")
}
