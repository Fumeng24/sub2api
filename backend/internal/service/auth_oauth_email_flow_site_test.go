//go:build unit

package service

import (
	"context"
)

func (s *redeemCodeRepoStub) ListByIDs(context.Context, []int64) ([]RedeemCode, error) {
	panic("unexpected ListByIDs call")
}
