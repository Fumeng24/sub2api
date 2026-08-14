package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPrepareUsageLogInsert_PersistsVideoMetadata(t *testing.T) {
	resolution := "720p"
	duration := 10
	prepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:               1,
		APIKeyID:             2,
		AccountID:            3,
		RequestID:            "req-video-metadata",
		Model:                "grok-4-0709",
		RequestedModel:       "grok-4-0709",
		VideoCount:           1,
		VideoResolution:      &resolution,
		VideoDurationSeconds: &duration,
		CreatedAt:            time.Date(2025, 1, 7, 12, 0, 0, 0, time.UTC),
	})

	require.Equal(t, 1, prepared.args[43])
	require.Equal(t, sql.NullString{String: resolution, Valid: true}, prepared.args[44])
	require.Equal(t, sql.NullInt64{Int64: int64(duration), Valid: true}, prepared.args[45])
}
