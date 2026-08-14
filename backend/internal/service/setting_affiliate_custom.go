package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	defaultGroupRateDiscountName      = "限时折扣"
	groupRateDiscountScheduleOnce     = "once"
	groupRateDiscountScheduleWeekly   = "weekly"
	groupRateDiscountSettingsCacheTTL = 10 * time.Second
)

type cachedGroupRateDiscountSettings struct {
	settings  GroupRateDiscountSettings
	expiresAt int64
}

// GetAffiliateBindBonusAmount returns the manually claimed invitee bonus.
func (s *SettingService) GetAffiliateBindBonusAmount(ctx context.Context) float64 {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyAffiliateBindBonusAmount)
	if err != nil {
		return AffiliateBindBonusAmountDefault
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return AffiliateBindBonusAmountDefault
	}
	return amount
}

func defaultGroupRateDiscountSettings() GroupRateDiscountSettings {
	return GroupRateDiscountSettings{
		Name:               defaultGroupRateDiscountName,
		DiscountMultiplier: 1,
		ScheduleMode:       groupRateDiscountScheduleWeekly,
		Weekdays:           []int{1, 2, 3, 4, 5, 6, 7},
		DailyStartTime:     "00:00",
		DailyEndTime:       "23:59",
		GroupIDs:           []int64{},
	}
}

func parseGroupRateDiscountSettings(raw string) GroupRateDiscountSettings {
	if strings.TrimSpace(raw) == "" {
		return defaultGroupRateDiscountSettings()
	}
	var settings GroupRateDiscountSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return defaultGroupRateDiscountSettings()
	}
	return normalizeGroupRateDiscountSettings(settings)
}

func normalizeGroupRateDiscountSettings(settings GroupRateDiscountSettings) GroupRateDiscountSettings {
	settings.Name = strings.TrimSpace(settings.Name)
	if settings.Name == "" {
		settings.Name = defaultGroupRateDiscountName
	}
	if math.IsNaN(settings.DiscountMultiplier) || math.IsInf(settings.DiscountMultiplier, 0) || settings.DiscountMultiplier <= 0 {
		settings.DiscountMultiplier = 1
	}
	settings.ScheduleMode = normalizeGroupRateDiscountScheduleMode(settings.ScheduleMode, settings.StartAt, settings.EndAt)
	settings.StartAt = normalizeRFC3339TimeString(settings.StartAt)
	settings.EndAt = normalizeRFC3339TimeString(settings.EndAt)
	settings.Weekdays = normalizeGroupRateDiscountWeekdays(settings.Weekdays)
	settings.DailyStartTime = normalizeDailyTimeString(settings.DailyStartTime)
	settings.DailyEndTime = normalizeDailyTimeString(settings.DailyEndTime)
	if settings.DailyStartTime == "" {
		settings.DailyStartTime = "00:00"
	}
	if settings.DailyEndTime == "" {
		settings.DailyEndTime = "23:59"
	}

	seen := make(map[int64]struct{}, len(settings.GroupIDs))
	groupIDs := make([]int64, 0, len(settings.GroupIDs))
	for _, id := range settings.GroupIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		groupIDs = append(groupIDs, id)
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	settings.GroupIDs = groupIDs
	return settings
}

func normalizeGroupRateDiscountScheduleMode(raw, startAt, endAt string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case groupRateDiscountScheduleOnce:
		return groupRateDiscountScheduleOnce
	case groupRateDiscountScheduleWeekly:
		return groupRateDiscountScheduleWeekly
	default:
		if strings.TrimSpace(startAt) != "" || strings.TrimSpace(endAt) != "" {
			return groupRateDiscountScheduleOnce
		}
		return groupRateDiscountScheduleWeekly
	}
}

func normalizeGroupRateDiscountWeekdays(raw []int) []int {
	seen := make(map[int]struct{}, len(raw))
	out := make([]int, 0, len(raw))
	for _, day := range raw {
		if day < 1 || day > 7 {
			continue
		}
		if _, ok := seen[day]; ok {
			continue
		}
		seen[day] = struct{}{}
		out = append(out, day)
	}
	sort.Ints(out)
	return out
}

func normalizeDailyTimeString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := time.Parse("15:04", raw)
	if err != nil {
		return raw
	}
	return parsed.Format("15:04")
}

func groupRateDiscountWeeklyActiveAt(settings GroupRateDiscountSettings, now time.Time) bool {
	if len(settings.Weekdays) == 0 {
		return false
	}
	startMinutes, ok := dailyTimeMinutes(settings.DailyStartTime)
	if !ok {
		return false
	}
	endMinutes, ok := dailyTimeMinutes(settings.DailyEndTime)
	if !ok || startMinutes == endMinutes {
		return false
	}
	localNow := now.In(time.Local)
	currentMinutes := localNow.Hour()*60 + localNow.Minute()
	currentDay := isoWeekday(localNow.Weekday())
	if startMinutes < endMinutes {
		return intSliceContains(settings.Weekdays, currentDay) && currentMinutes >= startMinutes && currentMinutes < endMinutes
	}
	if intSliceContains(settings.Weekdays, currentDay) && currentMinutes >= startMinutes {
		return true
	}
	previousDay := currentDay - 1
	if previousDay == 0 {
		previousDay = 7
	}
	return intSliceContains(settings.Weekdays, previousDay) && currentMinutes < endMinutes
}

func dailyTimeMinutes(value string) (int, bool) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func isoWeekday(day time.Weekday) int {
	if day == time.Sunday {
		return 7
	}
	return int(day)
}

func intSliceContains(values []int, needle int) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func normalizeRFC3339TimeString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return parsed.UTC().Format(time.RFC3339)
}

func (s *SettingService) GetGroupRateDiscountSettings(ctx context.Context) GroupRateDiscountSettings {
	if s == nil || s.settingRepo == nil {
		return defaultGroupRateDiscountSettings()
	}
	now := time.Now()
	if cached, ok := s.groupRateDiscountCache.Load().(*cachedGroupRateDiscountSettings); ok && now.UnixNano() < cached.expiresAt {
		return cached.settings
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyGroupRateDiscountSettings)
	settings := defaultGroupRateDiscountSettings()
	if err == nil {
		settings = parseGroupRateDiscountSettings(raw)
	}
	s.groupRateDiscountCache.Store(&cachedGroupRateDiscountSettings{
		settings:  settings,
		expiresAt: now.Add(groupRateDiscountSettingsCacheTTL).UnixNano(),
	})
	return settings
}

func (s *SettingService) ActiveGroupRateDiscount(ctx context.Context, now time.Time) *ActiveGroupRateDiscount {
	return s.GetGroupRateDiscountSettings(ctx).ActiveAt(now)
}

func (s *SettingService) validateGroupRateDiscountSettings(ctx context.Context, settings *GroupRateDiscountSettings) error {
	if settings == nil {
		return nil
	}
	normalized := normalizeGroupRateDiscountSettings(*settings)
	*settings = normalized
	if !settings.Enabled {
		return nil
	}
	if settings.DiscountMultiplier <= 0 || settings.DiscountMultiplier >= 1 {
		return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_MULTIPLIER", "discount multiplier must be greater than 0 and less than 1")
	}
	if len(settings.GroupIDs) == 0 {
		return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_GROUPS", "at least one group must be selected when group rate discount is enabled")
	}
	if settings.ScheduleMode == groupRateDiscountScheduleWeekly {
		if len(settings.Weekdays) == 0 {
			return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_WEEKDAYS", "at least one weekday must be selected when weekly group rate discount is enabled")
		}
		if _, ok := dailyTimeMinutes(settings.DailyStartTime); !ok {
			return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_DAILY_START", "daily_start_time must use HH:mm format")
		}
		if _, ok := dailyTimeMinutes(settings.DailyEndTime); !ok {
			return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_DAILY_END", "daily_end_time must use HH:mm format")
		}
		if settings.DailyStartTime == settings.DailyEndTime {
			return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_DAILY_WINDOW", "daily_end_time must differ from daily_start_time")
		}
	} else {
		start, err := time.Parse(time.RFC3339, settings.StartAt)
		if err != nil {
			return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_START", "start_at must be a valid RFC3339 timestamp")
		}
		end, err := time.Parse(time.RFC3339, settings.EndAt)
		if err != nil {
			return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_END", "end_at must be a valid RFC3339 timestamp")
		}
		if !end.After(start) {
			return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_WINDOW", "end_at must be after start_at")
		}
	}
	if s.defaultSubGroupReader == nil {
		return nil
	}
	for _, groupID := range settings.GroupIDs {
		if _, err := s.defaultSubGroupReader.GetByID(ctx, groupID); err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return infraerrors.BadRequest("INVALID_GROUP_RATE_DISCOUNT_GROUP", "discount group must exist").WithMetadata(map[string]string{
					"group_id": strconv.FormatInt(groupID, 10),
				})
			}
			return err
		}
	}
	return nil
}

func publicGroupRateDiscountState(raw string, now time.Time) (*ActiveGroupRateDiscount, *ActiveGroupRateDiscount) {
	settings := parseGroupRateDiscountSettings(raw)
	active := settings.ActiveAt(now)
	if active != nil {
		return active, nil
	}
	return nil, settings.PreviewAt(now)
}
