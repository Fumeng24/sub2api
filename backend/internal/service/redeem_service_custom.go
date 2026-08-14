package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type redeemCodeRepositoryCustom interface {
	ListByIDs(ctx context.Context, ids []int64) ([]RedeemCode, error)
}

type RedeemCodeBatchUpdateFieldsCustom struct {
	BusinessCategory *string
}

func SetRedeemCodeBatchBusinessCategoryCustom(fields *RedeemCodeBatchUpdateFields, category *string) {
	if fields != nil {
		fields.BusinessCategory = category
	}
}

func (f RedeemCodeBatchUpdateFields) hasCustomChanges() bool {
	return f.BusinessCategory != nil
}

func applyGeneratedRedeemCodeCustom(code *RedeemCode) {
	code.BusinessCategory = redeemCodeDefaultBusinessCategory(code.Type)
}

func prepareRedeemCodeBusinessCategoryCustom(code *RedeemCode) error {
	if code.BusinessCategory == "" {
		code.BusinessCategory = redeemCodeDefaultBusinessCategory(code.Type)
	}
	return ValidateRedeemCodeBusinessCategory(code.Type, code.Value, code.BusinessCategory)
}

func redeemCodeDefaultBusinessCategory(codeType string) string {
	if codeType == RedeemTypeBalance {
		return BalanceBusinessCategoryRecharge
	}
	return BalanceBusinessCategoryUnclassified
}

func (s *RedeemService) validateBatchBusinessCategoryCustom(ctx context.Context, ids []int64, fields *RedeemCodeBatchUpdateFields) error {
	if fields == nil || fields.BusinessCategory == nil {
		return nil
	}
	category := NormalizeBalanceBusinessCategory(*fields.BusinessCategory)
	if category != "" && !IsKnownBalanceBusinessCategory(category) {
		return infraerrors.BadRequest("REDEEM_CODE_BUSINESS_CATEGORY_INVALID", "business_category is invalid")
	}
	codes, err := s.redeemRepo.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(codes) != len(ids) {
		return ErrRedeemCodeNotFound
	}
	for _, code := range codes {
		if err := ValidateRedeemCodeBusinessCategory(code.Type, code.Value, category); err != nil {
			return err
		}
	}
	fields.BusinessCategory = &category
	return nil
}
