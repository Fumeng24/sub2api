package service

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

type systemSettingsCustom struct {
	AffiliateBindBonusAmount  float64
	GroupRateDiscountSettings GroupRateDiscountSettings `json:"group_rate_discount_settings"`
	TicketSystemConfig        TicketSystemSettings      `json:"ticket_system_config"`
}

type publicSettingsCustom struct {
	PaymentBalanceRechargeMultiplier float64
	GroupRateDiscount                *ActiveGroupRateDiscount `json:"group_rate_discount,omitempty"`
	UpcomingGroupRateDiscount        *ActiveGroupRateDiscount `json:"upcoming_group_rate_discount,omitempty"`
}

type GroupRateDiscountSettings struct {
	Enabled            bool    `json:"enabled"`
	Name               string  `json:"name"`
	DiscountMultiplier float64 `json:"discount_multiplier"`
	ScheduleMode       string  `json:"schedule_mode"`
	StartAt            string  `json:"start_at"`
	EndAt              string  `json:"end_at"`
	Weekdays           []int   `json:"weekdays"`
	DailyStartTime     string  `json:"daily_start_time"`
	DailyEndTime       string  `json:"daily_end_time"`
	GroupIDs           []int64 `json:"group_ids"`
}

type ActiveGroupRateDiscount struct {
	Name               string  `json:"name"`
	DiscountMultiplier float64 `json:"discount_multiplier"`
	ScheduleMode       string  `json:"schedule_mode"`
	StartAt            string  `json:"start_at"`
	EndAt              string  `json:"end_at"`
	Weekdays           []int   `json:"weekdays"`
	DailyStartTime     string  `json:"daily_start_time"`
	DailyEndTime       string  `json:"daily_end_time"`
	Timezone           string  `json:"timezone"`
	GroupIDs           []int64 `json:"group_ids"`
}

func (d *ActiveGroupRateDiscount) AppliesToGroup(groupID int64) bool {
	if d == nil || groupID <= 0 {
		return false
	}
	for _, id := range d.GroupIDs {
		if id == groupID {
			return true
		}
	}
	return false
}

func (s GroupRateDiscountSettings) ActiveAt(now time.Time) *ActiveGroupRateDiscount {
	normalized := normalizeGroupRateDiscountSettings(s)
	if !normalized.Enabled || normalized.DiscountMultiplier <= 0 || normalized.DiscountMultiplier >= 1 || len(normalized.GroupIDs) == 0 {
		return nil
	}
	if normalized.ScheduleMode == groupRateDiscountScheduleWeekly {
		if !groupRateDiscountWeeklyActiveAt(normalized, now) {
			return nil
		}
	} else {
		start, err := time.Parse(time.RFC3339, normalized.StartAt)
		if err != nil {
			return nil
		}
		end, err := time.Parse(time.RFC3339, normalized.EndAt)
		if err != nil || now.Before(start) || !now.Before(end) {
			return nil
		}
	}
	return normalized.toPublicGroupRateDiscount()
}

func (s GroupRateDiscountSettings) PreviewAt(now time.Time) *ActiveGroupRateDiscount {
	normalized := normalizeGroupRateDiscountSettings(s)
	if !normalized.Enabled || normalized.DiscountMultiplier <= 0 || normalized.DiscountMultiplier >= 1 || len(normalized.GroupIDs) == 0 {
		return nil
	}
	if normalized.ScheduleMode == groupRateDiscountScheduleWeekly {
		if len(normalized.Weekdays) == 0 || normalized.DailyStartTime == "" || normalized.DailyEndTime == "" || normalized.DailyStartTime == normalized.DailyEndTime {
			return nil
		}
		return normalized.toPublicGroupRateDiscount()
	}
	start, err := time.Parse(time.RFC3339, normalized.StartAt)
	if err != nil {
		return nil
	}
	end, err := time.Parse(time.RFC3339, normalized.EndAt)
	if err != nil || !start.Before(end) || !now.Before(end) {
		return nil
	}
	return normalized.toPublicGroupRateDiscount()
}

func (s GroupRateDiscountSettings) toPublicGroupRateDiscount() *ActiveGroupRateDiscount {
	return &ActiveGroupRateDiscount{
		Name: s.Name, DiscountMultiplier: s.DiscountMultiplier, ScheduleMode: s.ScheduleMode,
		StartAt: s.StartAt, EndAt: s.EndAt, Weekdays: append([]int(nil), s.Weekdays...),
		DailyStartTime: s.DailyStartTime, DailyEndTime: s.DailyEndTime, Timezone: timezone.Name(),
		GroupIDs: append([]int64(nil), s.GroupIDs...),
	}
}
