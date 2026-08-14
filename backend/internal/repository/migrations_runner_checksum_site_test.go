package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsMigrationChecksumCompatible_SiteHistorical155Checksum(t *testing.T) {
	ok := isMigrationChecksumCompatible(
		"155_invoice_requests.sql",
		"f4fc2eba77594ad2ba77303dc965e0f5f27ae9144310e6bb0d69d6fddbbabd20",
		"d401e5393189fc8e57f5a74aa7aca6ca23537fb5a73673b9c16a81c9927dd52f",
	)
	require.True(t, ok)
}
