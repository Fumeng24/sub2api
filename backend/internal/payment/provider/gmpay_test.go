package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestGMPaySignMatchesDocsExample(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"pid":          "1000",
		"order_id":     "ORD202605230001",
		"currency":     "cny",
		"token":        "usdt",
		"network":      "tron",
		"amount":       "100",
		"notify_url":   "https://merchant.example/notify",
		"redirect_url": "https://merchant.example/return",
		"name":         "VIP",
	}

	got := gmpaySign(params, "epusdt_secret_key")
	if got != "476412c422f4dd75c3d533f5c47a9cac" {
		t.Fatalf("gmpaySign() = %q", got)
	}
}

func TestGMPayCreatePaymentPostsSignedForm(t *testing.T) {
	t.Parallel()

	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != gmpayCreateTransactionPath {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		form = r.PostForm
		writeGMPayJSON(t, w, map[string]any{
			"status_code": 200,
			"message":     "success",
			"data": map[string]any{
				"trade_id":    "trade_123",
				"payment_url": serverURL(r) + "/cashier/trade_123",
			},
		})
	}))
	defer server.Close()

	prov, err := NewGMPay("1", map[string]string{
		"apiBase":   server.URL,
		"pid":       "1000",
		"secretKey": "secret",
		"notifyUrl": "https://app.example/api/v1/payment/webhook/gmpay",
		"currency":  "cny",
		"token":     "usdt",
		"network":   "tron",
	})
	if err != nil {
		t.Fatalf("NewGMPay: %v", err)
	}

	resp, err := prov.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:   "sub2_20260530abc12345",
		Amount:    "100.00",
		Subject:   "VIP",
		ReturnURL: "https://app.example/payment/result",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}

	if resp.TradeNo != "trade_123" {
		t.Fatalf("TradeNo = %q", resp.TradeNo)
	}
	if resp.Currency != "CNY" {
		t.Fatalf("Currency = %q", resp.Currency)
	}
	if !strings.Contains(resp.PayURL, "/cashier/trade_123") {
		t.Fatalf("PayURL = %q", resp.PayURL)
	}
	if form.Get("amount") != "100.00" {
		t.Fatalf("amount = %q", form.Get("amount"))
	}
	values := map[string]string{}
	for key := range form {
		values[key] = form.Get(key)
	}
	if form.Get("signature") != gmpaySign(values, "secret") {
		t.Fatalf("signature = %q", form.Get("signature"))
	}
}

func TestGMPayVerifyNotification(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"pid":                  "1000",
		"trade_id":             "trade_123",
		"order_id":             "sub2_20260530abc12345",
		"amount":               "100",
		"actual_amount":        "14.29",
		"receive_address":      "TTestTronAddress001",
		"token":                "USDT",
		"block_transaction_id": "0xabc123",
		"status":               "2",
	}
	params["signature"] = gmpaySign(params, "secret")
	raw := `{"pid":"1000","trade_id":"trade_123","order_id":"sub2_20260530abc12345","amount":100,"actual_amount":14.29,"receive_address":"TTestTronAddress001","token":"USDT","block_transaction_id":"0xabc123","signature":"` + params["signature"] + `","status":2}`

	prov, err := NewGMPay("1", map[string]string{
		"apiBase":   "https://pay.example.com",
		"pid":       "1000",
		"secretKey": "secret",
		"notifyUrl": "https://app.example/api/v1/payment/webhook/gmpay",
	})
	if err != nil {
		t.Fatalf("NewGMPay: %v", err)
	}
	notification, err := prov.VerifyNotification(context.Background(), raw, nil)
	if err != nil {
		t.Fatalf("VerifyNotification: %v", err)
	}
	if notification.OrderID != "sub2_20260530abc12345" || notification.TradeNo != "trade_123" {
		t.Fatalf("notification ids = %#v", notification)
	}
	if notification.Amount != 100 {
		t.Fatalf("notification amount = %v", notification.Amount)
	}
	if notification.Metadata["pid"] != "1000" || notification.Metadata["currency"] != "CNY" {
		t.Fatalf("metadata = %#v", notification.Metadata)
	}
}

func TestGMPayQueryOrderFetchesAmountWhenPaid(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case gmpayCheckStatusPath + "/trade_123":
			writeGMPayJSON(t, w, map[string]any{
				"status_code": 200,
				"message":     "success",
				"data": map[string]any{
					"trade_id": "trade_123",
					"status":   2,
				},
			})
		case gmpayCheckoutRespPath + "/trade_123":
			writeGMPayJSON(t, w, map[string]any{
				"status_code": 200,
				"message":     "success",
				"data": map[string]any{
					"trade_id":        "trade_123",
					"amount":          100,
					"actual_amount":   14.29,
					"token":           "USDT",
					"currency":        "CNY",
					"receive_address": "TTestTronAddress001",
					"network":         "tron",
				},
			})
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	prov, err := NewGMPay("1", map[string]string{
		"apiBase":   server.URL,
		"pid":       "1000",
		"secretKey": "secret",
		"notifyUrl": "https://app.example/api/v1/payment/webhook/gmpay",
	})
	if err != nil {
		t.Fatalf("NewGMPay: %v", err)
	}
	resp, err := prov.QueryOrder(context.Background(), "trade_123")
	if err != nil {
		t.Fatalf("QueryOrder: %v", err)
	}
	if resp.Status != payment.ProviderStatusPaid || resp.Amount != 100 {
		t.Fatalf("QueryOrder = %#v", resp)
	}
	if resp.Metadata["token"] != "USDT" || resp.Metadata["network"] != "tron" {
		t.Fatalf("metadata = %#v", resp.Metadata)
	}
}

func writeGMPayJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
