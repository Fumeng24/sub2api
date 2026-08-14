package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSortOrderBoundaries(t *testing.T) {
	require.NoError(t, validateSortOrder(monitorMinSortOrder))
	require.NoError(t, validateSortOrder(monitorMaxSortOrder))
	require.ErrorIs(t, validateSortOrder(monitorMinSortOrder-1), ErrChannelMonitorInvalidSortOrder)
	require.ErrorIs(t, validateSortOrder(monitorMaxSortOrder+1), ErrChannelMonitorInvalidSortOrder)
}
