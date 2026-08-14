package service

import (
	"context"
	"fmt"
	"math"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const minGroupRelativeRateMultiplier = 0.0001

// GroupRelativeRateMultiplierInput stores a coefficient relative to a group's
// standard rate. For example, 0.8 charges 80% and 1.2 charges 120%.
type GroupRelativeRateMultiplierInput struct {
	UserID     int64   `json:"user_id"`
	Multiplier float64 `json:"multiplier"`
}

// GroupRelativeRateMultiplierEntry is the admin-facing view of a user's
// relative coefficient. A fixed final rate is returned solely to expose a
// conflict: fixed rates take precedence over relative coefficients.
type GroupRelativeRateMultiplierEntry struct {
	UserID                 int64    `json:"user_id"`
	UserName               string   `json:"user_name"`
	UserEmail              string   `json:"user_email"`
	UserNotes              string   `json:"user_notes"`
	UserStatus             string   `json:"user_status"`
	RelativeRateMultiplier *float64 `json:"relative_rate_multiplier,omitempty"`
	FixedRateMultiplier    *float64 `json:"fixed_rate_multiplier,omitempty"`
}

type groupRelativeRateRepository interface {
	SyncGroupRelativeRateMultipliers(ctx context.Context, groupID int64, entries []GroupRelativeRateMultiplierInput) error
}

// GetGroupRelativeRateMultipliers returns relative coefficients along with
// fixed-rate conflicts for the selected group.
func (s *adminServiceImpl) GetGroupRelativeRateMultipliers(
	ctx context.Context,
	groupID int64,
) ([]GroupRelativeRateMultiplierEntry, error) {
	if err := s.ensureRelativeRateGroup(ctx, groupID); err != nil {
		return nil, err
	}
	if s.userGroupRateRepo == nil {
		return []GroupRelativeRateMultiplierEntry{}, nil
	}

	entries, err := s.userGroupRateRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]GroupRelativeRateMultiplierEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.DiscountMultiplier == nil && entry.RateMultiplier == nil {
			continue
		}
		out = append(out, GroupRelativeRateMultiplierEntry{
			UserID:                 entry.UserID,
			UserName:               entry.UserName,
			UserEmail:              entry.UserEmail,
			UserNotes:              entry.UserNotes,
			UserStatus:             entry.UserStatus,
			RelativeRateMultiplier: entry.DiscountMultiplier,
			FixedRateMultiplier:    entry.RateMultiplier,
		})
	}
	return out, nil
}

// SyncGroupRelativeRateMultipliers atomically replaces coefficients for one
// group and invalidates both gateway caches once the write has committed.
func (s *adminServiceImpl) SyncGroupRelativeRateMultipliers(
	ctx context.Context,
	groupID int64,
	entries []GroupRelativeRateMultiplierInput,
) error {
	if err := s.ensureRelativeRateGroup(ctx, groupID); err != nil {
		return err
	}
	if s.userGroupRateRepo == nil {
		return infraerrors.InternalServer("RELATIVE_RATE_UNAVAILABLE", "relative rate repository is not configured")
	}

	seenUserIDs := make(map[int64]struct{}, len(entries))
	for _, entry := range entries {
		if entry.UserID <= 0 {
			return infraerrors.BadRequest("INVALID_USER_ID", "user_id must be positive")
		}
		if _, exists := seenUserIDs[entry.UserID]; exists {
			return infraerrors.BadRequest("DUPLICATE_USER_ID", fmt.Sprintf("duplicate user_id: %d", entry.UserID))
		}
		seenUserIDs[entry.UserID] = struct{}{}
		if entry.Multiplier < minGroupRelativeRateMultiplier || entry.Multiplier > 100 || math.IsNaN(entry.Multiplier) || math.IsInf(entry.Multiplier, 0) {
			return infraerrors.BadRequest("INVALID_RELATIVE_MULTIPLIER", fmt.Sprintf("multiplier must be between %.4f and 100 (user_id=%d)", minGroupRelativeRateMultiplier, entry.UserID))
		}
		if _, err := s.userRepo.GetByID(ctx, entry.UserID); err != nil {
			return err
		}
	}

	existing, err := s.userGroupRateRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return err
	}
	for _, entry := range existing {
		if entry.RateMultiplier == nil {
			continue
		}
		if _, selected := seenUserIDs[entry.UserID]; selected {
			return infraerrors.Conflict(
				"FIXED_RATE_CONFLICT",
				fmt.Sprintf("user %d has a fixed final rate for this group; remove it before setting a relative multiplier", entry.UserID),
			)
		}
	}

	repo, ok := s.userGroupRateRepo.(groupRelativeRateRepository)
	if !ok {
		return infraerrors.InternalServer("RELATIVE_RATE_UNAVAILABLE", "relative rate repository is not configured")
	}
	if err := repo.SyncGroupRelativeRateMultipliers(ctx, groupID, entries); err != nil {
		return err
	}
	s.invalidateUserGroupRateCaches(groupID)
	return nil
}

func (s *adminServiceImpl) ensureRelativeRateGroup(ctx context.Context, groupID int64) error {
	if groupID <= 0 {
		return infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be positive")
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return err
	}
	// Subscription groups are governed by their plan quotas rather than the
	// balance-billed group multiplier. Keeping relative coefficients on standard
	// groups avoids storing settings which cannot affect subscription billing.
	if group.SubscriptionType != SubscriptionTypeStandard {
		return infraerrors.BadRequest("UNSUPPORTED_GROUP_TYPE", "relative rate multipliers are only supported for standard groups")
	}
	return nil
}

func (s *adminServiceImpl) invalidateUserGroupRateCaches(groupID int64) {
	if s.gatewayService != nil {
		s.gatewayService.InvalidateUserGroupRateCache(groupID)
	}
	if s.openAIGatewayService != nil {
		s.openAIGatewayService.InvalidateUserGroupRateCache(groupID)
	}
}
