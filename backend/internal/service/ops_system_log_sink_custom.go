package service

import "github.com/Wei-Shaw/sub2api/internal/pkg/logger"

func shouldIndexAccessLogCustom(event *logger.LogEvent) bool {
	if event == nil || event.Fields == nil {
		return false
	}
	status := asInt64Ptr(event.Fields["status_code"])
	return status != nil && *status >= 400
}

func asInt64PtrCustom(v any) (*int64, bool) {
	var n int64
	switch t := v.(type) {
	case int32:
		n = int64(t)
	case int16:
		n = int64(t)
	case int8:
		n = int64(t)
	case uint:
		n = int64(t)
	case uint64:
		if t > uint64(^uint64(0)>>1) {
			return nil, true
		}
		n = int64(t)
	case uint32:
		n = int64(t)
	case uint16:
		n = int64(t)
	case uint8:
		n = int64(t)
	case float32:
		n = int64(t)
	default:
		return nil, false
	}
	if n <= 0 {
		return nil, true
	}
	return &n, true
}
