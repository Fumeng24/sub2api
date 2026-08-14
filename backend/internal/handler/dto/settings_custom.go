package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

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

func GroupRateDiscountSettingsFromService(s service.GroupRateDiscountSettings) GroupRateDiscountSettings {
	return GroupRateDiscountSettings{
		Enabled: s.Enabled, Name: s.Name, DiscountMultiplier: s.DiscountMultiplier,
		ScheduleMode: s.ScheduleMode, StartAt: s.StartAt, EndAt: s.EndAt,
		Weekdays: append([]int{}, s.Weekdays...), DailyStartTime: s.DailyStartTime,
		DailyEndTime: s.DailyEndTime, GroupIDs: append([]int64{}, s.GroupIDs...),
	}
}

func GroupRateDiscountSettingsToService(s GroupRateDiscountSettings) service.GroupRateDiscountSettings {
	return service.GroupRateDiscountSettings{
		Enabled: s.Enabled, Name: s.Name, DiscountMultiplier: s.DiscountMultiplier,
		ScheduleMode: s.ScheduleMode, StartAt: s.StartAt, EndAt: s.EndAt,
		Weekdays: append([]int{}, s.Weekdays...), DailyStartTime: s.DailyStartTime,
		DailyEndTime: s.DailyEndTime, GroupIDs: append([]int64{}, s.GroupIDs...),
	}
}

func ActiveGroupRateDiscountFromService(s *service.ActiveGroupRateDiscount) *ActiveGroupRateDiscount {
	if s == nil {
		return nil
	}
	return &ActiveGroupRateDiscount{
		Name: s.Name, DiscountMultiplier: s.DiscountMultiplier, ScheduleMode: s.ScheduleMode,
		StartAt: s.StartAt, EndAt: s.EndAt, Weekdays: append([]int{}, s.Weekdays...),
		DailyStartTime: s.DailyStartTime, DailyEndTime: s.DailyEndTime, Timezone: s.Timezone,
		GroupIDs: append([]int64{}, s.GroupIDs...),
	}
}
