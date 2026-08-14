//go:build unit

package service

import (
	"context"
)

func (r *paymentFulfillmentAffiliateRepoStub) ClaimBindBonus(context.Context, int64, float64) (bool, float64, error) {
	panic("unexpected ClaimBindBonus call")
}
