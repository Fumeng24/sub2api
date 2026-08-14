package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeCustomOpenAIGroupFields(t *testing.T) {
	openAIGroup := &Group{Platform: PlatformOpenAI, ForceOpenAIPriority: true, OpenAIStableLowTTFT: true}
	sanitizeCustomOpenAIGroupFields(openAIGroup)
	require.True(t, openAIGroup.ForceOpenAIPriority)
	require.True(t, openAIGroup.OpenAIStableLowTTFT)

	otherGroup := &Group{Platform: PlatformAnthropic, ForceOpenAIPriority: true, OpenAIStableLowTTFT: true}
	sanitizeCustomOpenAIGroupFields(otherGroup)
	require.False(t, otherGroup.ForceOpenAIPriority)
	require.False(t, otherGroup.OpenAIStableLowTTFT)
}
