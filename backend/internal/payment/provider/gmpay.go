package provider

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const (
	gmpayCreateTransactionPath = "/payments/gmpay/v1/order/create-transaction"
	gmpayCheckoutPath          = "/pay/checkout-counter"
	gmpayCheckoutRespPath      = "/pay/checkout-counter-resp"
	gmpayCheckStatusPath       = "/pay/check-status"
	gmpayHTTPTimeout           = 10 * time.Second
	gmpayMaxResponseSize       = 1 << 20
	gmpayMaxErrorSummary       = 512
	gmpayStatusPending         = 1
	gmpayStatusPaid            = 2
	gmpayStatusExpired         = 3
	gmpayDefaultCurrency       = payment.DefaultPaymentCurrency
	gmpayDefaultToken          = "USDT"
	gmpayDefaultNetwork        = "tron"
)

// GMPay implements the Epusdt/GM Pay hosted crypto checkout API.
type GMPay struct {
	instanceID string
	config     map[string]string
	httpClient *http.Client
}

func NewGMPay(instanceID string, config map[string]string) (*GMPay, error) {
	for _, k := range []string{"apiBase", "pid", "secretKey", "notifyUrl"} {
		if strings.TrimSpace(config[k]) == "" {
			return nil, fmt.Errorf("gmpay config missing required key: %s", k)
		}
	}
	cfg := cloneStringMap(config)
	apiBase, err := normalizeGMPayAPIBase(cfg["apiBase"])
	if err != nil {
		return nil, err
	}
	cfg["apiBase"] = apiBase
	currency, err := payment.NormalizePaymentCurrency(cfg["currency"])
	if err != nil {
		return nil, fmt.Errorf("gmpay config currency: %w", err)
	}
	cfg["currency"] = currency
	if strings.TrimSpace(cfg["token"]) == "" {
		cfg["token"] = gmpayDefaultToken
	}
	cfg["token"] = strings.ToUpper(strings.TrimSpace(cfg["token"]))
	if strings.TrimSpace(cfg["network"]) == "" {
		cfg["network"] = gmpayDefaultNetwork
	}
	cfg["network"] = strings.ToLower(strings.TrimSpace(cfg["network"]))
	return &GMPay{
		instanceID: instanceID,
		config:     cfg,
		httpClient: &http.Client{Timeout: gmpayHTTPTimeout},
	}, nil
}

func normalizeGMPayAPIBase(raw string) (string, error) {
	base := strings.TrimSpace(raw)
	if base == "" {
		return "", fmt.Errorf("gmpay apiBase is required")
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("gmpay apiBase must be an absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("gmpay apiBase must use http or https")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.RawPath = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	for _, suffix := range []string{
		gmpayCreateTransactionPath,
		gmpayCheckoutPath,
		gmpayCheckoutRespPath,
		gmpayCheckStatusPath,
	} {
		if strings.HasSuffix(parsed.Path, suffix) {
			parsed.Path = strings.TrimRight(strings.TrimSuffix(parsed.Path, suffix), "/")
			break
		}
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func (g *GMPay) Name() string        { return "GM Pay" }
func (g *GMPay) ProviderKey() string { return payment.TypeGMPay }
func (g *GMPay) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeUSDT}
}

func (g *GMPay) MerchantIdentityMetadata() map[string]string {
	if g == nil {
		return nil
	}
	return map[string]string{
		"pid":      strings.TrimSpace(g.config["pid"]),
		"currency": g.currency(),
		"token":    strings.ToUpper(strings.TrimSpace(g.config["token"])),
		"network":  strings.ToLower(strings.TrimSpace(g.config["network"])),
	}
}

func (g *GMPay) currency() string {
	if g == nil {
		return gmpayDefaultCurrency
	}
	currency, err := payment.NormalizePaymentCurrency(g.config["currency"])
	if err != nil {
		return gmpayDefaultCurrency
	}
	return currency
}

func (g *GMPay) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("gmpay create payment: invalid amount %s", req.Amount)
	}

	notifyURL := strings.TrimSpace(req.NotifyURL)
	if notifyURL == "" {
		notifyURL = strings.TrimSpace(g.config["notifyUrl"])
	}
	if notifyURL == "" {
		return nil, fmt.Errorf("gmpay notifyUrl is required")
	}

	returnURL := strings.TrimSpace(req.ReturnURL)
	if returnURL == "" {
		returnURL = strings.TrimSpace(g.config["returnUrl"])
	}

	params := map[string]string{
		"pid":          strings.TrimSpace(g.config["pid"]),
		"order_id":     strings.TrimSpace(req.OrderID),
		"currency":     strings.ToLower(g.currency()),
		"token":        strings.ToLower(strings.TrimSpace(g.config["token"])),
		"network":      strings.ToLower(strings.TrimSpace(g.config["network"])),
		"amount":       amount.StringFixed(int32(payment.CurrencyMaxFractionDigits(g.currency()))),
		"notify_url":   notifyURL,
		"redirect_url": returnURL,
		"name":         strings.TrimSpace(req.Subject),
		"payment_type": payment.TypeUSDT,
	}
	params["signature"] = gmpaySign(params, g.config["secretKey"])

	var resp gmpayTransactionResponse
	if err := g.postForm(ctx, g.url(gmpayCreateTransactionPath), params, &resp); err != nil {
		return nil, fmt.Errorf("gmpay create payment: %w", err)
	}
	if err := resp.ok(); err != nil {
		return nil, fmt.Errorf("gmpay create payment: %w", err)
	}
	tradeID := strings.TrimSpace(resp.Data.TradeID)
	if tradeID == "" {
		return nil, fmt.Errorf("gmpay create payment: missing trade_id")
	}
	payURL := strings.TrimSpace(resp.Data.PaymentURL)
	if payURL == "" {
		payURL = g.url(gmpayCheckoutPath + "/" + url.PathEscape(tradeID))
	}
	return &payment.CreatePaymentResponse{
		TradeNo:  tradeID,
		PayURL:   payURL,
		Currency: g.currency(),
	}, nil
}

func (g *GMPay) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	tradeID := strings.TrimSpace(tradeNo)
	if tradeID == "" {
		return nil, fmt.Errorf("gmpay query order: missing trade_id")
	}

	var statusResp gmpayStatusResponse
	if err := g.getJSON(ctx, g.url(gmpayCheckStatusPath+"/"+url.PathEscape(tradeID)), &statusResp); err != nil {
		return nil, fmt.Errorf("gmpay query order: %w", err)
	}
	if err := statusResp.ok(); err != nil {
		return nil, fmt.Errorf("gmpay query order: %w", err)
	}

	status := gmpayProviderStatus(statusResp.Data.Status)
	amount := 0.0
	metadata := g.MerchantIdentityMetadata()
	metadata["status"] = strconv.Itoa(statusResp.Data.Status)
	if status == payment.ProviderStatusPaid {
		detail, err := g.checkoutDetail(ctx, tradeID)
		if err != nil {
			return nil, fmt.Errorf("gmpay query order detail: %w", err)
		}
		amount = detail.Amount.InexactFloat64()
		for k, v := range detail.metadata(g.currency()) {
			metadata[k] = v
		}
	}
	return &payment.QueryOrderResponse{
		TradeNo:  tradeID,
		Status:   status,
		Amount:   amount,
		Metadata: metadata,
	}, nil
}

func (g *GMPay) checkoutDetail(ctx context.Context, tradeID string) (*gmpayTransactionData, error) {
	var resp gmpayTransactionResponse
	if err := g.getJSON(ctx, g.url(gmpayCheckoutRespPath+"/"+url.PathEscape(tradeID)), &resp); err != nil {
		return nil, err
	}
	if err := resp.ok(); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (g *GMPay) VerifyNotification(_ context.Context, rawBody string, _ map[string]string) (*payment.PaymentNotification, error) {
	if strings.TrimSpace(rawBody) == "" {
		return nil, fmt.Errorf("gmpay notification empty body")
	}
	if err := verifyGMPayRawJSONSignature(rawBody, g.config["secretKey"]); err != nil {
		return nil, err
	}

	var payload gmpayNotificationPayload
	if err := json.Unmarshal([]byte(rawBody), &payload); err != nil {
		return nil, fmt.Errorf("gmpay parse notification: %w", err)
	}
	if strings.TrimSpace(payload.PID) == "" || strings.TrimSpace(payload.PID) != strings.TrimSpace(g.config["pid"]) {
		return nil, fmt.Errorf("gmpay pid mismatch")
	}
	if payload.Status != gmpayStatusPaid {
		return nil, nil
	}
	if strings.TrimSpace(payload.TradeID) == "" || strings.TrimSpace(payload.OrderID) == "" {
		return nil, fmt.Errorf("gmpay notification missing trade_id or order_id")
	}
	metadata := g.MerchantIdentityMetadata()
	metadata["actual_amount"] = payload.ActualAmount.String()
	metadata["receive_address"] = strings.TrimSpace(payload.ReceiveAddress)
	metadata["block_transaction_id"] = strings.TrimSpace(payload.BlockTransactionID)
	metadata["status"] = strconv.Itoa(payload.Status)
	if token := strings.TrimSpace(payload.Token); token != "" {
		metadata["token"] = strings.ToUpper(token)
	}
	return &payment.PaymentNotification{
		TradeNo:  strings.TrimSpace(payload.TradeID),
		OrderID:  strings.TrimSpace(payload.OrderID),
		Amount:   payload.Amount.InexactFloat64(),
		Status:   payment.NotificationStatusSuccess,
		RawData:  rawBody,
		Metadata: metadata,
	}, nil
}

func (g *GMPay) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, fmt.Errorf("gmpay refund is not supported")
}

func (g *GMPay) url(path string) string {
	return strings.TrimRight(g.config["apiBase"], "/") + path
}

func (g *GMPay) postForm(ctx context.Context, endpoint string, params map[string]string, out any) error {
	values := url.Values{}
	for k, v := range params {
		if strings.TrimSpace(v) != "" {
			values.Set(k, v)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return g.do(req, out)
}

func (g *GMPay) getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	return g.do(req, out)
}

func (g *GMPay) do(req *http.Request, out any) error {
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, gmpayMaxResponseSize))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, summarizeGMPayBody(body))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func summarizeGMPayBody(body []byte) string {
	body = bytes.TrimSpace(body)
	if len(body) > gmpayMaxErrorSummary {
		body = body[:gmpayMaxErrorSummary]
	}
	return string(body)
}

func gmpaySign(params map[string]string, secretKey string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "signature" || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+params[key])
	}
	sum := md5.Sum([]byte(strings.Join(pairs, "&") + secretKey))
	return hex.EncodeToString(sum[:])
}

func verifyGMPayRawJSONSignature(rawBody string, secretKey string) error {
	values, err := gmpayRawJSONSignValues(rawBody)
	if err != nil {
		return err
	}
	signature := strings.TrimSpace(values["signature"])
	if signature == "" {
		return fmt.Errorf("gmpay notification missing signature")
	}
	expected := gmpaySign(values, secretKey)
	if !strings.EqualFold(signature, expected) {
		return fmt.Errorf("gmpay signature mismatch")
	}
	return nil
}

func gmpayRawJSONSignValues(rawBody string) (map[string]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(rawBody), &raw); err != nil {
		return nil, fmt.Errorf("gmpay parse signature json: %w", err)
	}
	values := make(map[string]string, len(raw))
	for key, message := range raw {
		trimmed := bytes.TrimSpace(message)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			continue
		}
		if bytes.Equal(trimmed, []byte(`""`)) {
			continue
		}
		if len(trimmed) > 0 && trimmed[0] == '"' {
			var s string
			if err := json.Unmarshal(trimmed, &s); err != nil {
				return nil, fmt.Errorf("gmpay parse signature field %s: %w", key, err)
			}
			if s == "" {
				continue
			}
			values[key] = s
			continue
		}
		values[key] = string(trimmed)
	}
	return values, nil
}

func gmpayProviderStatus(status int) string {
	switch status {
	case gmpayStatusPaid:
		return payment.ProviderStatusPaid
	case gmpayStatusExpired:
		return payment.ProviderStatusFailed
	default:
		return payment.ProviderStatusPending
	}
}

type gmpayBaseResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
}

func (r gmpayBaseResponse) ok() error {
	if r.StatusCode == 0 || r.StatusCode == 200 {
		return nil
	}
	msg := strings.TrimSpace(r.Message)
	if msg == "" {
		msg = "request failed"
	}
	return fmt.Errorf("status_code %d: %s", r.StatusCode, msg)
}

type gmpayTransactionResponse struct {
	gmpayBaseResponse
	Data gmpayTransactionData `json:"data"`
}

func (r gmpayTransactionResponse) ok() error {
	return r.gmpayBaseResponse.ok()
}

type gmpayStatusResponse struct {
	gmpayBaseResponse
	Data struct {
		TradeID string `json:"trade_id"`
		Status  int    `json:"status"`
	} `json:"data"`
}

func (r gmpayStatusResponse) ok() error {
	return r.gmpayBaseResponse.ok()
}

type gmpayTransactionData struct {
	TradeID        string          `json:"trade_id"`
	Amount         decimal.Decimal `json:"amount"`
	ActualAmount   decimal.Decimal `json:"actual_amount"`
	Token          string          `json:"token"`
	Currency       string          `json:"currency"`
	ReceiveAddress string          `json:"receive_address"`
	Network        string          `json:"network"`
	RedirectURL    string          `json:"redirect_url"`
	PaymentURL     string          `json:"payment_url"`
	Status         int             `json:"status"`
}

func (d gmpayTransactionData) metadata(defaultCurrency string) map[string]string {
	currency := strings.ToUpper(strings.TrimSpace(d.Currency))
	if currency == "" {
		currency = defaultCurrency
	}
	return map[string]string{
		"currency":        currency,
		"token":           strings.ToUpper(strings.TrimSpace(d.Token)),
		"network":         strings.ToLower(strings.TrimSpace(d.Network)),
		"actual_amount":   d.ActualAmount.String(),
		"receive_address": strings.TrimSpace(d.ReceiveAddress),
	}
}

type gmpayNotificationPayload struct {
	PID                string          `json:"pid"`
	TradeID            string          `json:"trade_id"`
	OrderID            string          `json:"order_id"`
	Amount             decimal.Decimal `json:"amount"`
	ActualAmount       decimal.Decimal `json:"actual_amount"`
	ReceiveAddress     string          `json:"receive_address"`
	Token              string          `json:"token"`
	BlockTransactionID string          `json:"block_transaction_id"`
	Signature          string          `json:"signature"`
	Status             int             `json:"status"`
}

var _ payment.MerchantIdentityProvider = (*GMPay)(nil)
