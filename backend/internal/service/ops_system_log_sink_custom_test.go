package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestOpsSystemLogSinkCustom_AccessStatusTypes(t *testing.T) {
	require.True(t, shouldIndexAccessLogCustom(&logger.LogEvent{
		Fields: map[string]any{"status_code": uint32(503)},
	}))
	require.False(t, shouldIndexAccessLogCustom(&logger.LogEvent{
		Fields: map[string]any{"status_code": float32(200)},
	}))
}

func TestAsInt64PtrCustom_ExtendedTypes(t *testing.T) {
	for _, input := range []any{int32(7), int16(7), int8(7), uint(7), uint64(7), uint32(7), uint16(7), uint8(7), float32(7)} {
		got := asInt64Ptr(input)
		require.NotNil(t, got, "input type %T", input)
		require.Equal(t, int64(7), *got)
	}
	require.Nil(t, asInt64Ptr(int32(0)))
	require.Nil(t, asInt64Ptr(^uint64(0)))
}
