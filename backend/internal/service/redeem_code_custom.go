package service

type RedeemCodeCustom struct {
	BusinessCategory string
}

func SetRedeemCodeBusinessCategoryCustom(code *RedeemCode, category string) {
	if code != nil {
		code.BusinessCategory = category
	}
}
