package admin

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupRequestCustomFieldsBindAndApply(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		var req CreateGroupRequest
		require.NoError(t, json.Unmarshal([]byte(`{
			"name":"priority",
			"force_openai_priority":true,
			"openai_stable_low_ttft":true,
			"auto_sort_config":{"enabled":true,"basis":"latency"}
		}`), &req))

		input := &service.CreateGroupInput{}
		applyCreateGroupInputCustom(&req, input)
		require.True(t, input.ForceOpenAIPriority)
		require.True(t, input.OpenAIStableLowTTFT)
		require.True(t, input.AutoSortConfig.Enabled)
		require.Equal(t, "latency", input.AutoSortConfig.Basis)
	})

	t.Run("update", func(t *testing.T) {
		var req UpdateGroupRequest
		require.NoError(t, json.Unmarshal([]byte(`{
			"force_openai_priority":false,
			"openai_stable_low_ttft":true,
			"auto_sort_config":{"enabled":true,"basis":"success_rate"}
		}`), &req))

		input := &service.UpdateGroupInput{}
		applyUpdateGroupInputCustom(&req, input)
		require.NotNil(t, input.ForceOpenAIPriority)
		require.False(t, *input.ForceOpenAIPriority)
		require.NotNil(t, input.OpenAIStableLowTTFT)
		require.True(t, *input.OpenAIStableLowTTFT)
		require.NotNil(t, input.AutoSortConfig)
		require.True(t, input.AutoSortConfig.Enabled)
		require.Equal(t, "success_rate", input.AutoSortConfig.Basis)
	})
}
