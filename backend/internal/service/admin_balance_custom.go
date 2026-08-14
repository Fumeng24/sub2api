package service

import "context"

type AdminBalanceCategoryService interface {
	UpdateUserBalanceWithCategory(ctx context.Context, userID int64, balance float64, operation, notes, businessCategory string) (*User, error)
}

func UpdateAdminUserBalance(ctx context.Context, admin AdminService, userID int64, balance float64, operation, notes, businessCategory string) (*User, error) {
	if extended, ok := admin.(AdminBalanceCategoryService); ok {
		return extended.UpdateUserBalanceWithCategory(ctx, userID, balance, operation, notes, businessCategory)
	}
	return admin.UpdateUserBalance(ctx, userID, balance, operation, notes)
}
