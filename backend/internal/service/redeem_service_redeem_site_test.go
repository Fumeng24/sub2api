package service

import (
	"context"
)

func (r *redeemRejectRepo) ListByIDs(ctx context.Context, ids []int64) ([]RedeemCode, error) {
	panic("unexpected ListByIDs call")
}
