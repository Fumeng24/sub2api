package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnnouncementTargeting_Matches_EmptyMatchesAll(t *testing.T) {
	var targeting AnnouncementTargeting
	require.True(t, targeting.Matches(0, nil))
	require.True(t, targeting.Matches(123.45, map[int64]struct{}{1: {}}))
}

func TestAnnouncementTargeting_NormalizeAndValidate_RejectsEmptyGroup(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{AllOf: nil},
		},
	}
	_, err := targeting.NormalizeAndValidate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAnnouncementInvalidTarget)
}

func TestAnnouncementTargeting_NormalizeAndValidate_RejectsInvalidCondition(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{
				AllOf: []AnnouncementCondition{
					{Type: "balance", Operator: "between", Value: 10},
				},
			},
		},
	}
	_, err := targeting.NormalizeAndValidate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAnnouncementInvalidTarget)
}

func TestAnnouncementTargeting_Matches_AndOrSemantics(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{
			{
				AllOf: []AnnouncementCondition{
					{Type: AnnouncementConditionTypeBalance, Operator: AnnouncementOperatorGTE, Value: 100},
					{Type: AnnouncementConditionTypeSubscription, Operator: AnnouncementOperatorIn, GroupIDs: []int64{10}},
				},
			},
			{
				AllOf: []AnnouncementCondition{
					{Type: AnnouncementConditionTypeBalance, Operator: AnnouncementOperatorLT, Value: 5},
				},
			},
		},
	}

	// 命中第 2 组（balance < 5）
	require.True(t, targeting.Matches(4.99, nil))
	require.False(t, targeting.Matches(5, nil))

	// 命中第 1 组（balance >= 100 AND 订阅 in [10]）
	require.False(t, targeting.Matches(100, map[int64]struct{}{}))
	require.False(t, targeting.Matches(99.9, map[int64]struct{}{10: {}}))
	require.True(t, targeting.Matches(100, map[int64]struct{}{10: {}}))
}

func TestAnnouncementTargeting_MatchesForUser(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{{
			AllOf: []AnnouncementCondition{{
				Type:     AnnouncementConditionTypeUser,
				Operator: AnnouncementOperatorIn,
				UserIDs:  []int64{42, 84},
			}},
		}},
	}

	require.True(t, targeting.MatchesForUser(42, 0, nil))
	require.False(t, targeting.MatchesForUser(7, 0, nil))
	require.False(t, targeting.Matches(0, nil), "user-only targeting must not match without a user ID")
}

func TestAnnouncementTargeting_NormalizeAndValidatePreservesUserIDs(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{{
			AllOf: []AnnouncementCondition{{
				Type:     AnnouncementConditionTypeUser,
				Operator: AnnouncementOperatorIn,
				UserIDs:  []int64{42, 84},
			}},
		}},
	}

	normalized, err := targeting.NormalizeAndValidate()
	require.NoError(t, err)
	require.Equal(t, []int64{42, 84}, normalized.AnyOf[0].AllOf[0].UserIDs)
}

func TestAnnouncementTargeting_NormalizeAndValidateUserIDs(t *testing.T) {
	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{{
			AllOf: []AnnouncementCondition{{
				Type:     AnnouncementConditionTypeUser,
				Operator: AnnouncementOperatorIn,
				UserIDs:  []int64{0},
			}},
		}},
	}
	_, err := targeting.NormalizeAndValidate()
	require.ErrorIs(t, err, ErrAnnouncementInvalidTarget)
}

func TestAnnouncementTargeting_NormalizeAndValidateRejectsTooManyUserIDs(t *testing.T) {
	userIDs := make([]int64, 101)
	for i := range userIDs {
		userIDs[i] = int64(i + 1)
	}

	targeting := AnnouncementTargeting{
		AnyOf: []AnnouncementConditionGroup{{
			AllOf: []AnnouncementCondition{{
				Type:     AnnouncementConditionTypeUser,
				Operator: AnnouncementOperatorIn,
				UserIDs:  userIDs,
			}},
		}},
	}

	_, err := targeting.NormalizeAndValidate()
	require.ErrorIs(t, err, ErrAnnouncementInvalidTarget)
}
