//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func (s *userGroupRateRepoStubForGroupRate) GetByUserAndGroup(context.Context, int64, int64) (*float64, error) {
	panic("unexpected GetByUserAndGroup call")
}

func (s *userGroupRateRepoStubForListUsers) GetByUserAndGroup(context.Context, int64, int64) (*float64, error) {
	panic("unexpected GetByUserAndGroup call")
}

func TestUserGroupRateEntryCustomFieldsRemainFlattenedInJSON(t *testing.T) {
	discount := 0.8
	body, err := json.Marshal(UserGroupRateEntry{
		UserID: 7,
		userGroupRateEntryCustom: userGroupRateEntryCustom{
			DiscountMultiplier: &discount,
		},
	})
	require.NoError(t, err)
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &payload))
	require.JSONEq(t, `0.8`, string(payload["discount_multiplier"]))
	require.NotContains(t, payload, "userGroupRateEntryCustom")
}
