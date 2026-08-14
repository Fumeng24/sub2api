package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *adminServiceImpl) tryUpdateUserBalanceCustom(ctx context.Context, userID int64, balance float64, operation, notes string) (*User, error, bool) {
	user, err := s.updateUserBalanceCustom(ctx, userID, balance, operation, notes, "")
	return user, err, true
}

func (s *adminServiceImpl) updateUserBalanceCustom(ctx context.Context, userID int64, balance float64, operation string, notes string, businessCategory string) (*User, error) {
	if operation != "set" && operation != "add" && operation != "subtract" {
		return nil, errors.New("invalid operation: must be 'set', 'add', or 'subtract'")
	}

	var user *User
	var balanceDiff float64

	applyAdjustment := func(opCtx context.Context) error {
		var change BalanceChange
		var err error
		switch operation {
		case "set":
			change, err = s.userRepo.SetBalance(opCtx, userID, balance)
		case "add":
			change, err = s.userRepo.AdjustBalance(opCtx, userID, balance)
		case "subtract":
			change, err = s.userRepo.AdjustBalance(opCtx, userID, -balance)
		}
		if errors.Is(err, ErrBalanceNegative) {
			return fmt.Errorf("balance cannot be negative, current balance: %.2f, requested operation would result in: %.2f", change.Old, change.New)
		}
		if err != nil {
			return err
		}
		balanceDiff = change.New - change.Old
		category, err := NormalizeAdminBalanceBusinessCategory(operation, balanceDiff, businessCategory)
		if err != nil {
			return err
		}
		user, err = s.userRepo.GetByID(opCtx, userID)
		if err != nil {
			return err
		}
		if balanceDiff == 0 {
			return nil
		}
		code, err := GenerateRedeemCode()
		if err != nil {
			return fmt.Errorf("generate balance adjustment redeem code: %w", err)
		}

		adjustmentRecord := &RedeemCode{
			Code:   code,
			Type:   AdjustmentTypeAdminBalance,
			Value:  balanceDiff,
			Status: StatusUsed,
			UsedBy: &user.ID,
			RedeemCodeCustom: RedeemCodeCustom{
				BusinessCategory: category,
			},
			Notes: notes,
		}
		now := time.Now()
		adjustmentRecord.UsedAt = &now

		if err := s.redeemCodeRepo.Create(opCtx, adjustmentRecord); err != nil {
			return fmt.Errorf("create balance adjustment redeem code: %w", err)
		}
		return nil
	}

	if s.entClient != nil {
		tx, err := s.entClient.Tx(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback() }()
		if err := applyAdjustment(dbent.NewTxContext(ctx, tx)); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	} else if err := applyAdjustment(ctx); err != nil {
		return nil, err
	}

	if s.authCacheInvalidator != nil && balanceDiff != 0 {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	s.tryAccrueAffiliateRebateForAdminRecharge(ctx, userID, operation, balance)
	if s.billingCacheService != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, userID); err != nil {
				logger.LegacyPrintf("service.admin", "invalidate user balance cache failed: user_id=%d err=%v", userID, err)
			}
		}()
	}

	return user, nil
}

func (s *adminServiceImpl) UpdateUserBalanceWithCategory(ctx context.Context, userID int64, balance float64, operation, notes, businessCategory string) (*User, error) {
	return s.updateUserBalanceCustom(ctx, userID, balance, operation, notes, businessCategory)
}
