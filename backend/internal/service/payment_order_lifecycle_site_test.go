//go:build unit

package service

import (
	"context"
	_ "modernc.org/sqlite"
)

func (r *paymentOrderLifecycleRedeemRepo) ListByIDs(context.Context, []int64) ([]RedeemCode, error) {
	panic("unexpected call")
}
