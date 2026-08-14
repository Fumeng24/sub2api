//go:build unit

package handler

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorUserOverviewResponseCustom(t *testing.T) {
	checkedAt := time.Date(2026, 7, 14, 22, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	latency := 125
	overview := &service.UserMonitorOverview{
		Items: []*service.UserMonitorView{{
			ID: 7, Name: "Codex", Provider: "openai", PrimaryModel: "gpt-5.6",
			PrimaryStatus: "operational", PrimaryLatencyMs: &latency,
		}},
		LastUpdatedAt: &checkedAt,
		TrendPeriod:   "7d",
	}

	got := channelMonitorUserOverviewResponseCustom(overview)
	require.Len(t, got.Items, 1)
	require.Equal(t, int64(7), got.Items[0].ID)
	require.Equal(t, "gpt-5.6", got.Items[0].PrimaryModel)
	require.NotNil(t, got.LastUpdatedAt)
	require.Equal(t, "2026-07-14T14:30:00Z", *got.LastUpdatedAt)
	require.Equal(t, "7d", got.TrendPeriod)
}
