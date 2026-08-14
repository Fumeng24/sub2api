package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/ent/redeemcode"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func applyRedeemCodeEntityCustom(out *service.RedeemCode, model *dbent.RedeemCode) {
	out.BusinessCategory = model.BusinessCategory
}

func positiveBalanceTypePredicateCustom() predicate.RedeemCode {
	return redeemcode.Or(
		redeemcode.TypeEQ(service.RedeemTypeBalance),
		redeemcode.And(
			redeemcode.TypeEQ(service.AdjustmentTypeAdminBalance),
			redeemcode.BusinessCategoryIn(
				service.BalanceBusinessCategoryManualCollection,
				service.BalanceBusinessCategoryManualRefund,
			),
		),
	)
}

func applyRedeemCodeBatchBusinessCategoryCustom(up *dbent.RedeemCodeUpdate, fields service.RedeemCodeBatchUpdateFields) {
	if fields.BusinessCategory != nil {
		up.SetBusinessCategory(*fields.BusinessCategory)
	}
}

func (r *redeemCodeRepository) ListByIDs(ctx context.Context, ids []int64) ([]service.RedeemCode, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	client := clientFromContext(ctx, r.client)
	codes, err := client.RedeemCode.Query().Where(redeemcode.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil, err
	}
	return redeemCodeEntitiesToService(codes), nil
}
