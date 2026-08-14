package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyFromService_MapsCategory(t *testing.T) {
	src := &service.APIKey{Category: service.APIKeyCategoryAnthropic}

	out := APIKeyFromService(src)

	require.NotNil(t, out)
	require.Equal(t, service.APIKeyCategoryAnthropic, out.Category)
}
