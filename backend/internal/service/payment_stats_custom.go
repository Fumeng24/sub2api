package service

import (
	"context"
	"sort"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/redeemcode"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

const ignoredPaymentDashboardUserID int64 = 1

type paymentDashboardRecordCustom struct {
	UserID      int64
	UserEmail   string
	Amount      float64
	Currency    string
	Count       int
	PaymentType string
	OccurredAt  time.Time
}

func (s *PaymentService) getDashboardStatsCustom(ctx context.Context, days int) (*DashboardStats, bool, error) {
	if days <= 0 {
		days = 30
	}
	now := time.Now()
	since := now.AddDate(0, 0, -days)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	records, err := s.listPaymentDashboardRecordsCustom(ctx)
	if err != nil {
		return nil, true, err
	}

	stats := &DashboardStats{}
	computeBasicStatsCustom(stats, records, todayStart)
	stats.PendingOrders, err = s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusEQ(OrderStatusPending),
			paymentorder.UserIDNEQ(ignoredPaymentDashboardUserID),
		).
		Count(ctx)
	if err != nil {
		return nil, true, err
	}

	stats.DailySeries = buildDailySeriesCustom(records, since, days)
	stats.PaymentMethods = buildMethodDistributionCustom(records)
	stats.TopUsers = buildTopUsersCustom(records)
	return stats, true, nil
}

func (s *PaymentService) listPaymentDashboardRecordsCustom(ctx context.Context) ([]paymentDashboardRecordCustom, error) {
	paidStatuses := []string{
		OrderStatusCompleted,
		OrderStatusPaid,
		OrderStatusRecharging,
		OrderStatusPartiallyRefunded,
		OrderStatusRefunded,
	}
	orders, err := s.entClient.PaymentOrder.Query().
		Where(paymentorder.StatusIn(paidStatuses...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	records := make([]paymentDashboardRecordCustom, 0, len(orders)*2)
	for _, order := range orders {
		if order.UserID == ignoredPaymentDashboardUserID {
			continue
		}
		currency := PaymentOrderCurrency(order)
		records = append(records, paymentDashboardRecordCustom{
			UserID:      order.UserID,
			UserEmail:   order.UserEmail,
			Amount:      order.PayAmount,
			Currency:    currency,
			Count:       1,
			PaymentType: order.PaymentType,
			OccurredAt:  paymentOrderDashboardTimeCustom(order),
		})
		if order.RefundAmount > 0 && paymentOrderDashboardRefundedCustom(order) {
			refundAmount := calculateGatewayRefundAmount(
				order.Amount,
				order.PayAmount,
				paymentOrderDashboardRefundAmountCustom(order),
				currency,
			)
			records = append(records, paymentDashboardRecordCustom{
				UserID:      order.UserID,
				UserEmail:   order.UserEmail,
				Amount:      -refundAmount,
				Currency:    currency,
				Count:       0,
				PaymentType: order.PaymentType,
				OccurredAt:  paymentOrderDashboardRefundTimeCustom(order),
			})
		}
	}

	paymentCodeSet, err := s.paymentOrderRechargeCodeSetCustom(ctx)
	if err != nil {
		return nil, err
	}
	redeems, err := s.entClient.RedeemCode.Query().
		Where(
			redeemcode.StatusEQ(StatusUsed),
			redeemcode.UsedByNotNil(),
			redeemcode.Or(
				redeemcode.TypeEQ(RedeemTypeBalance),
				redeemcode.And(
					redeemcode.TypeEQ(AdjustmentTypeAdminBalance),
					redeemcode.BusinessCategoryIn(
						BalanceBusinessCategoryManualCollection,
						BalanceBusinessCategoryManualRefund,
					),
				),
			),
		).
		WithUser().
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, redeem := range redeems {
		if redeem.UsedBy != nil && *redeem.UsedBy == ignoredPaymentDashboardUserID {
			continue
		}
		if paymentCodeSet[redeem.Code] {
			continue
		}
		userID := int64(0)
		if redeem.UsedBy != nil {
			userID = *redeem.UsedBy
		}
		userEmail := ""
		if user := redeem.Edges.User; user != nil {
			userEmail = user.Email
		}
		paymentType := "redeem_code"
		if redeem.Type == AdjustmentTypeAdminBalance {
			paymentType = "admin_balance"
		}
		records = append(records, paymentDashboardRecordCustom{
			UserID:      userID,
			UserEmail:   userEmail,
			Amount:      redeem.Value,
			Currency:    payment.DefaultPaymentCurrency,
			Count:       1,
			PaymentType: paymentType,
			OccurredAt:  redeemCodeDashboardTimeCustom(redeem),
		})
	}

	return records, nil
}

func (s *PaymentService) paymentOrderRechargeCodeSetCustom(ctx context.Context) (map[string]bool, error) {
	codes, err := s.entClient.PaymentOrder.Query().
		Select(paymentorder.FieldRechargeCode).
		Strings(ctx)
	if err != nil {
		return nil, err
	}
	codeSet := make(map[string]bool, len(codes))
	for _, code := range codes {
		if code != "" {
			codeSet[code] = true
		}
	}
	return codeSet, nil
}

func paymentOrderDashboardRefundedCustom(order *dbent.PaymentOrder) bool {
	return order.Status == OrderStatusRefunded || order.Status == OrderStatusPartiallyRefunded
}

func paymentOrderDashboardRefundAmountCustom(order *dbent.PaymentOrder) float64 {
	if order.RefundAmount > order.Amount {
		return order.Amount
	}
	return order.RefundAmount
}

func paymentOrderDashboardTimeCustom(order *dbent.PaymentOrder) time.Time {
	if order.PaidAt != nil {
		return *order.PaidAt
	}
	if order.CompletedAt != nil {
		return *order.CompletedAt
	}
	return order.CreatedAt
}

func paymentOrderDashboardRefundTimeCustom(order *dbent.PaymentOrder) time.Time {
	if order.RefundAt != nil {
		return *order.RefundAt
	}
	return order.UpdatedAt
}

func redeemCodeDashboardTimeCustom(redeem *dbent.RedeemCode) time.Time {
	if redeem.UsedAt != nil {
		return *redeem.UsedAt
	}
	return redeem.CreatedAt
}

func computeBasicStatsCustom(stats *DashboardStats, records []paymentDashboardRecordCustom, todayStart time.Time) {
	stats.TotalAmount = make(CurrencyAmounts)
	stats.TodayAmount = make(CurrencyAmounts)
	stats.AvgAmount = make(CurrencyAmounts)
	currencyCounts := make(map[string]int)
	var totalCount, todayCount int
	for _, record := range records {
		stats.TotalAmount[record.Currency] += record.Amount
		currencyCounts[record.Currency] += record.Count
		totalCount += record.Count
		if !record.OccurredAt.Before(todayStart) {
			stats.TodayAmount[record.Currency] += record.Amount
			todayCount += record.Count
		}
	}
	stats.TotalCount = totalCount
	stats.TodayCount = todayCount
	for currency, totalAmount := range stats.TotalAmount {
		if currencyCounts[currency] > 0 {
			stats.AvgAmount[currency] = roundAmount(totalAmount / float64(currencyCounts[currency]))
		}
	}
	roundCurrencyAmounts(stats.TotalAmount)
	roundCurrencyAmounts(stats.TodayAmount)
}

func buildDailySeriesCustom(records []paymentDashboardRecordCustom, since time.Time, days int) []DailyStats {
	dailyMap := make(map[string]*DailyStats)
	for _, record := range records {
		date := record.OccurredAt.Format("2006-01-02")
		stats, ok := dailyMap[date]
		if !ok {
			stats = &DailyStats{Date: date, Amount: make(CurrencyAmounts)}
			dailyMap[date] = stats
		}
		stats.Amount[record.Currency] += record.Amount
		stats.Count += record.Count
	}
	series := make([]DailyStats, 0, days)
	for i := 0; i < days; i++ {
		date := since.AddDate(0, 0, i+1).Format("2006-01-02")
		if stats, ok := dailyMap[date]; ok {
			roundCurrencyAmounts(stats.Amount)
			series = append(series, *stats)
		} else {
			series = append(series, DailyStats{Date: date, Amount: make(CurrencyAmounts)})
		}
	}
	return series
}

func buildMethodDistributionCustom(records []paymentDashboardRecordCustom) []PaymentMethodStat {
	methodMap := make(map[string]*PaymentMethodStat)
	for _, record := range records {
		stats, ok := methodMap[record.PaymentType]
		if !ok {
			stats = &PaymentMethodStat{Type: record.PaymentType, Amount: make(CurrencyAmounts)}
			methodMap[record.PaymentType] = stats
		}
		stats.Amount[record.Currency] += record.Amount
		stats.Count += record.Count
	}
	methods := make([]PaymentMethodStat, 0, len(methodMap))
	for _, stats := range methodMap {
		roundCurrencyAmounts(stats.Amount)
		methods = append(methods, *stats)
	}
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Type < methods[j].Type
	})
	return methods
}

func buildTopUsersCustom(records []paymentDashboardRecordCustom) TopUsersByCurrency {
	userMap := make(map[string]map[int64]*TopUserStat)
	for _, record := range records {
		users, ok := userMap[record.Currency]
		if !ok {
			users = make(map[int64]*TopUserStat)
			userMap[record.Currency] = users
		}
		stats, ok := users[record.UserID]
		if !ok {
			stats = &TopUserStat{UserID: record.UserID, Email: record.UserEmail}
			users[record.UserID] = stats
		}
		if stats.Email == "" {
			stats.Email = record.UserEmail
		}
		stats.Amount += record.Amount
	}
	result := make(TopUsersByCurrency, len(userMap))
	for currency, users := range userMap {
		userList := make([]*TopUserStat, 0, len(users))
		for _, stats := range users {
			stats.Amount = roundAmount(stats.Amount)
			userList = append(userList, stats)
		}
		sort.Slice(userList, func(i, j int) bool {
			return userList[i].Amount > userList[j].Amount
		})
		limit := topUsersLimit
		if len(userList) < limit {
			limit = len(userList)
		}
		result[currency] = make([]TopUserStat, 0, limit)
		for i := 0; i < limit; i++ {
			result[currency] = append(result[currency], *userList[i])
		}
	}
	return result
}
