package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelMonitorRuntimeAllowsPassiveV2AlongsideV1(t *testing.T) {
	v1 := ChannelMonitorRuntime{Enabled: true, Mode: ChannelMonitorModeV1}
	require.True(t, v1.ActiveProbesAllowed())
	require.True(t, v1.PassiveAggregationAllowed())

	v2 := ChannelMonitorRuntime{Enabled: true, Mode: ChannelMonitorModeV2}
	require.False(t, v2.ActiveProbesAllowed())
	require.True(t, v2.PassiveAggregationAllowed())

	disabled := ChannelMonitorRuntime{Enabled: false, Mode: ChannelMonitorModeV1}
	require.False(t, disabled.ActiveProbesAllowed())
	require.False(t, disabled.PassiveAggregationAllowed())
}
