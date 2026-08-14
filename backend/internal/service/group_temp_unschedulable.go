package service

import (
	"strconv"
	"strings"
	"time"
)

const groupTempUnschedulableKey = "group_temp_unschedulable"

func (a *Account) IsGroupTempUnschedulableAt(groupID int64, now time.Time) bool {
	until := a.groupTempUnschedulableUntil(groupID)
	return until != nil && now.Before(*until)
}

func (a *Account) groupTempUnschedulableUntil(groupID int64) *time.Time {
	if a == nil || a.Extra == nil || groupID <= 0 {
		return nil
	}
	rawBlocks, ok := a.Extra[groupTempUnschedulableKey].(map[string]any)
	if !ok {
		return nil
	}
	rawBlock, ok := rawBlocks[strconv.FormatInt(groupID, 10)].(map[string]any)
	if !ok {
		return nil
	}
	rawUntil, ok := rawBlock["until"].(string)
	if !ok || strings.TrimSpace(rawUntil) == "" {
		return nil
	}
	until, err := time.Parse(time.RFC3339, rawUntil)
	if err != nil {
		return nil
	}
	return &until
}
