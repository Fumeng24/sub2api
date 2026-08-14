package service

import infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

const (
	monitorMinSortOrder = 0
	monitorMaxSortOrder = 100000

	monitorProviderChallengeMaxTokens = 50
	monitorPlaceholderSessionID       = "{{monitor_session_id}}"
	monitorPlaceholderDeviceID        = "{{monitor_device_id}}"
)

var (
	ErrChannelMonitorInvalidSortOrder = infraerrors.BadRequest(
		"CHANNEL_MONITOR_INVALID_SORT_ORDER", "sort_order must be in [0, 100000]",
	)
	ErrChannelMonitorInvalidAPIKeyID = infraerrors.BadRequest(
		"CHANNEL_MONITOR_INVALID_API_KEY_ID", "api_key_id must be a positive integer",
	)
)
