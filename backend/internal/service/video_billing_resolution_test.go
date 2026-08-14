package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeVideoBillingResolutionOrDefault(t *testing.T) {
	require.Equal(t, VideoBillingResolution480P, NormalizeVideoBillingResolutionOrDefault(""))
	require.Equal(t, VideoBillingResolution480P, NormalizeVideoBillingResolutionOrDefault("sd"))
	require.Equal(t, VideoBillingResolution720P, NormalizeVideoBillingResolutionOrDefault("hd"))
	require.Equal(t, VideoBillingResolution1080P, NormalizeVideoBillingResolutionOrDefault("full_hd"))
	require.Equal(t, VideoBillingResolution1080P, NormalizeVideoBillingResolutionOrDefault("fhd"))
	require.Equal(t, VideoBillingResolution480P, NormalizeVideoBillingResolutionOrDefault("unknown"))
}

func TestNormalizeVideoBillingDurationSecondsOrDefault(t *testing.T) {
	require.Equal(t, VideoBillingDefaultDurationSeconds, NormalizeVideoBillingDurationSecondsOrDefault(0))
	require.Equal(t, VideoBillingMinDurationSeconds, NormalizeVideoBillingDurationSecondsOrDefault(1))
	require.Equal(t, 10, NormalizeVideoBillingDurationSecondsOrDefault(10))
	require.Equal(t, VideoBillingMaxDurationSeconds, NormalizeVideoBillingDurationSecondsOrDefault(999))
}
