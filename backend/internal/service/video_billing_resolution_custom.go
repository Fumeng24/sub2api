package service

import (
	"strconv"
	"strings"
)

func ParseVideoBillingDurationSeconds(value string) int {
	duration, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return duration
}
