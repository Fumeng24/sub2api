package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractUpstreamModelIDs_SiteStringArrayCompatibility(t *testing.T) {
	got, err := extractUpstreamModelIDs([]byte(`{"models":["gpt-5","models/gemini-2.5-flash","gpt-5",""]}`))
	require.NoError(t, err)
	require.Equal(t, []string{"gemini-2.5-flash", "gpt-5"}, got)
}
