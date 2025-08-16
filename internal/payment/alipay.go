package payment

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/pkg/errors"
)

// AlipayProvider implements PaymentProvider for Alipay
type AlipayProvider struct {
	AppID      string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Gateway    string
	NotifyURL  string
	ReturnURL  string
}

// AlipayConfig holds Alipay configuration
type AlipayConfig struct {
	AppID          string `json:"app_id"`
	PrivateKeyPath string `json:"private_key_path"`
	PublicKeyPath  string `json:"public_key_path"`
	Gateway        string `json:"gateway"`
	NotifyURL      string `json:"notify_url"`
	ReturnURL      string `json:"return_url"`
}

// NewAlipayProvider creates a new Alipay payment provider
func NewAlipayProvider(config AlipayConfig) (*AlipayProvider, error) {
	privateKey, err := loadRSAPrivateKey(config.PrivateKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load private key")
	}

	publicKey, err := loadRSAPublicKey(config.PublicKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load public key")
	}

	if config.Gateway == "" {
		config.Gateway = "https://openapi.alipay.com/gateway.do"
	}

	return &AlipayProvider{
		AppID:      config.AppID,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Gateway:    config.Gateway,
		NotifyURL:  config.NotifyURL,
		ReturnURL:  config.ReturnURL,
	}, nil
}

// CreateOrder creates an Alipay payment order
func (ap *AlipayProvider) CreateOrder(order *model.PaymentOrder) (*PaymentResponse, error) {
	// Build request parameters
	params := map[string]string{
		"app_id":      ap.AppID,
		"method":      "alipay.trade.precreate",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  ap.NotifyURL,
		"return_url":  ap.ReturnURL,
	}

	// Build business parameters
	bizContent := map[string]interface{}{
		"out_trade_no": order.OrderNo,
		"total_amount": fmt.Sprintf("%.2f", float64(order.Amount)/100),
		"subject":      fmt.Sprintf("OpenList Credits Purchase - %d credits", order.Credits),
		"body":         fmt.Sprintf("Purchase %d credits for OpenList", order.Credits),
		"timeout_express": "30m",
	}

	bizContentJSON, err := json.Marshal(bizContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal biz_content")
	}
	params["biz_content"] = string(bizContentJSON)

	// Generate signature
	sign, err := ap.generateSign(params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate signature")
	}
	params["sign"] = sign

	// Make API request
	resp, err := ap.makeAPIRequest(params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make API request")
	}

	// Parse response
	var alipayResp struct {
		AlipayTradePrecreateResponse struct {
			Code       string `json:"code"`
			Msg        string `json:"msg"`
			SubCode    string `json:"sub_code"`
			SubMsg     string `json:"sub_msg"`
			OutTradeNo string `json:"out_trade_no"`
			QRCode     string `json:"qr_code"`
		} `json:"alipay_trade_precreate_response"`
		Sign string `json:"sign"`
	}

	if err := json.Unmarshal(resp, &alipayResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}

	if alipayResp.AlipayTradePrecreateResponse.Code != "10000" {
		return nil, errors.Errorf("alipay error: %s - %s", 
			alipayResp.AlipayTradePrecreateResponse.Code,
			alipayResp.AlipayTradePrecreateResponse.Msg)
	}

	return &PaymentResponse{
		OrderNo: order.OrderNo,
		QRCode:  alipayResp.AlipayTradePrecreateResponse.QRCode,
		PaymentData: map[string]interface{}{
			"provider":    "alipay",
			"qr_code":     alipayResp.AlipayTradePrecreateResponse.QRCode,
			"out_trade_no": alipayResp.AlipayTradePrecreateResponse.OutTradeNo,
		},
	}, nil
}

// VerifyPayment verifies an Alipay payment notification
func (ap *AlipayProvider) VerifyPayment(orderNo string, paymentData map[string]interface{}) (*PaymentVerification, error) {
	// Extract notification parameters
	notifyParams := make(map[string]string)
	for key, value := range paymentData {
		if str, ok := value.(string); ok {
			notifyParams[key] = str
		}
	}

	// Verify signature
	if !ap.verifyNotifySign(notifyParams) {
		return &PaymentVerification{Success: false}, errors.New("invalid signature")
	}

	// Check trade status
	tradeStatus := notifyParams["trade_status"]
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		return &PaymentVerification{Success: false}, errors.New("payment not successful")
	}

	// Parse amount
	var amount float64
	if amountStr, exists := notifyParams["total_amount"]; exists {
		fmt.Sscanf(amountStr, "%f", &amount)
	}

	// Parse paid time
	paidAt := time.Now()
	if gmtPayment, exists := notifyParams["gmt_payment"]; exists {
		if t, err := time.Parse("2006-01-02 15:04:05", gmtPayment); err == nil {
			paidAt = t
		}
	}

	return &PaymentVerification{
		Success:       true,
		OrderNo:       notifyParams["out_trade_no"],
		TransactionID: notifyParams["trade_no"],
		Amount:        amount,
		PaidAt:        paidAt,
		PaymentData:   paymentData,
	}, nil
}

// Refund processes a refund for Alipay payment
func (ap *AlipayProvider) Refund(orderNo string, amount float64) (*RefundResponse, error) {
	// Build request parameters
	params := map[string]string{
		"app_id":    ap.AppID,
		"method":    "alipay.trade.refund",
		"charset":   "utf-8",
		"sign_type": "RSA2",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"version":   "1.0",
	}

	// Build business parameters
	bizContent := map[string]interface{}{
		"out_trade_no":   orderNo,
		"refund_amount":  fmt.Sprintf("%.2f", amount),
		"refund_reason":  "User requested refund",
		"out_request_no": fmt.Sprintf("%s_refund_%d", orderNo, time.Now().Unix()),
	}

	bizContentJSON, err := json.Marshal(bizContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal biz_content")
	}
	params["biz_content"] = string(bizContentJSON)

	// Generate signature
	sign, err := ap.generateSign(params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate signature")
	}
	params["sign"] = sign

	// Make API request
	resp, err := ap.makeAPIRequest(params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make API request")
	}

	// Parse response
	var alipayResp struct {
		AlipayTradeRefundResponse struct {
			Code         string `json:"code"`
			Msg          string `json:"msg"`
			OutTradeNo   string `json:"out_trade_no"`
			RefundFee    string `json:"refund_fee"`
			OutRequestNo string `json:"out_request_no"`
		} `json:"alipay_trade_refund_response"`
	}

	if err := json.Unmarshal(resp, &alipayResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}

	if alipayResp.AlipayTradeRefundResponse.Code != "10000" {
		return &RefundResponse{
			Success: false,
			Message: alipayResp.AlipayTradeRefundResponse.Msg,
		}, nil
	}

	return &RefundResponse{
		Success:  true,
		RefundID: alipayResp.AlipayTradeRefundResponse.OutRequestNo,
		Message:  "Refund successful",
	}, nil
}

// Helper methods

func (ap *AlipayProvider) generateSign(params map[string]string) (string, error) {
	// Remove sign parameter if exists
	delete(params, "sign")

	// Sort parameters
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build query string
	var query []string
	for _, key := range keys {
		if params[key] != "" {
			query = append(query, fmt.Sprintf("%s=%s", key, params[key]))
		}
	}
	queryString := strings.Join(query, "&")

	// Generate signature
	hash := sha256.Sum256([]byte(queryString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, ap.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

func (ap *AlipayProvider) verifyNotifySign(params map[string]string) bool {
	sign := params["sign"]
	delete(params, "sign")
	delete(params, "sign_type")

	// Sort parameters
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build query string
	var query []string
	for _, key := range keys {
		if params[key] != "" {
			query = append(query, fmt.Sprintf("%s=%s", key, params[key]))
		}
	}
	queryString := strings.Join(query, "&")

	// Verify signature
	signatureBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}

	hash := sha256.Sum256([]byte(queryString))
	err = rsa.VerifyPKCS1v15(ap.PublicKey, crypto.SHA256, hash[:], signatureBytes)
	return err == nil
}

func (ap *AlipayProvider) makeAPIRequest(params map[string]string) ([]byte, error) {
	// Build form data
	formData := url.Values{}
	for key, value := range params {
		formData.Set(key, value)
	}

	// Make HTTP request
	resp, err := http.PostForm(ap.Gateway, formData)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func loadRSAPrivateKey(keyPath string) (*rsa.PrivateKey, error) {
	// This is a placeholder implementation
	// In a real implementation, you would load the key from file
	return rsa.GenerateKey(rand.Reader, 2048)
}

func loadRSAPublicKey(keyPath string) (*rsa.PublicKey, error) {
	// This is a placeholder implementation
	// In a real implementation, you would load the key from file
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &privateKey.PublicKey, nil
}