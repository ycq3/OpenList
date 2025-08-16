package payment

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/pkg/errors"
)

// WechatProvider implements PaymentProvider for WeChat Pay
type WechatProvider struct {
	AppID     string
	MchID     string
	APIKey    string
	NotifyURL string
	Gateway   string
}

// WechatConfig holds WeChat Pay configuration
type WechatConfig struct {
	AppID     string `json:"app_id"`
	MchID     string `json:"mch_id"`
	APIKey    string `json:"api_key"`
	NotifyURL string `json:"notify_url"`
	Gateway   string `json:"gateway"`
}

// WechatUnifiedOrderRequest represents WeChat unified order request
type WechatUnifiedOrderRequest struct {
	XMLName        xml.Name `xml:"xml"`
	AppID          string   `xml:"appid"`
	MchID          string   `xml:"mch_id"`
	NonceStr       string   `xml:"nonce_str"`
	Sign           string   `xml:"sign"`
	Body           string   `xml:"body"`
	OutTradeNo     string   `xml:"out_trade_no"`
	TotalFee       int      `xml:"total_fee"`
	SpbillCreateIP string   `xml:"spbill_create_ip"`
	NotifyURL      string   `xml:"notify_url"`
	TradeType      string   `xml:"trade_type"`
}

// WechatUnifiedOrderResponse represents WeChat unified order response
type WechatUnifiedOrderResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
	ReturnMsg  string   `xml:"return_msg"`
	AppID      string   `xml:"appid"`
	MchID      string   `xml:"mch_id"`
	NonceStr   string   `xml:"nonce_str"`
	Sign       string   `xml:"sign"`
	ResultCode string   `xml:"result_code"`
	PrepayID   string   `xml:"prepay_id"`
	TradeType  string   `xml:"trade_type"`
	CodeURL    string   `xml:"code_url"`
	ErrCode    string   `xml:"err_code"`
	ErrCodeDes string   `xml:"err_code_des"`
}

// WechatNotification represents WeChat payment notification
type WechatNotification struct {
	XMLName       xml.Name `xml:"xml"`
	ReturnCode    string   `xml:"return_code"`
	ReturnMsg     string   `xml:"return_msg"`
	AppID         string   `xml:"appid"`
	MchID         string   `xml:"mch_id"`
	NonceStr      string   `xml:"nonce_str"`
	Sign          string   `xml:"sign"`
	ResultCode    string   `xml:"result_code"`
	OpenID        string   `xml:"openid"`
	TradeType     string   `xml:"trade_type"`
	BankType      string   `xml:"bank_type"`
	TotalFee      int      `xml:"total_fee"`
	TransactionID string   `xml:"transaction_id"`
	OutTradeNo    string   `xml:"out_trade_no"`
	TimeEnd       string   `xml:"time_end"`
}

// NewWechatProvider creates a new WeChat Pay provider
func NewWechatProvider(config WechatConfig) *WechatProvider {
	if config.Gateway == "" {
		config.Gateway = "https://api.mch.weixin.qq.com/pay/unifiedorder"
	}

	return &WechatProvider{
		AppID:     config.AppID,
		MchID:     config.MchID,
		APIKey:    config.APIKey,
		NotifyURL: config.NotifyURL,
		Gateway:   config.Gateway,
	}
}

// CreateOrder creates a WeChat Pay order
func (wp *WechatProvider) CreateOrder(order *model.PaymentOrder) (*PaymentResponse, error) {
	// Generate nonce string
	nonceStr := wp.generateNonceStr()

	// Build request
	req := WechatUnifiedOrderRequest{
		AppID:          wp.AppID,
		MchID:          wp.MchID,
		NonceStr:       nonceStr,
		Body:           fmt.Sprintf("OpenList Credits Purchase - %d credits", order.Credits),
		OutTradeNo:     order.OrderNo,
		TotalFee:       int(order.Amount * 100), // Convert to cents
		SpbillCreateIP: "127.0.0.1",
		NotifyURL:      wp.NotifyURL,
		TradeType:      "NATIVE", // QR code payment
	}

	// Generate signature
	req.Sign = wp.generateSign(req)

	// Convert to XML
	xmlData, err := xml.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	// Make API request
	resp, err := http.Post(wp.Gateway, "application/xml", strings.NewReader(string(xmlData)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to make API request")
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}

	// Parse response
	var wechatResp WechatUnifiedOrderResponse
	if err := xml.Unmarshal(respBody, &wechatResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse response")
	}

	// Check response
	if wechatResp.ReturnCode != "SUCCESS" {
		return nil, errors.Errorf("wechat error: %s", wechatResp.ReturnMsg)
	}

	if wechatResp.ResultCode != "SUCCESS" {
		return nil, errors.Errorf("wechat error: %s - %s", wechatResp.ErrCode, wechatResp.ErrCodeDes)
	}

	return &PaymentResponse{
		OrderNo: order.OrderNo,
		QRCode:  wechatResp.CodeURL,
		PaymentData: map[string]interface{}{
			"provider":   "wechat",
			"prepay_id":  wechatResp.PrepayID,
			"code_url":   wechatResp.CodeURL,
			"trade_type": wechatResp.TradeType,
		},
	}, nil
}

// VerifyPayment verifies a WeChat Pay notification
func (wp *WechatProvider) VerifyPayment(orderNo string, paymentData map[string]interface{}) (*PaymentVerification, error) {
	// Parse notification data
	notificationXML, ok := paymentData["xml"].(string)
	if !ok {
		return &PaymentVerification{Success: false}, errors.New("invalid notification data")
	}

	var notification WechatNotification
	if err := xml.Unmarshal([]byte(notificationXML), &notification); err != nil {
		return &PaymentVerification{Success: false}, errors.Wrap(err, "failed to parse notification")
	}

	// Verify signature
	if !wp.verifyNotificationSign(notification) {
		return &PaymentVerification{Success: false}, errors.New("invalid signature")
	}

	// Check payment status
	if notification.ReturnCode != "SUCCESS" || notification.ResultCode != "SUCCESS" {
		return &PaymentVerification{Success: false}, errors.New("payment not successful")
	}

	// Parse paid time
	paidAt := time.Now()
	if notification.TimeEnd != "" {
		if t, err := time.Parse("20060102150405", notification.TimeEnd); err == nil {
			paidAt = t
		}
	}

	return &PaymentVerification{
		Success:       true,
		OrderNo:       notification.OutTradeNo,
		TransactionID: notification.TransactionID,
		Amount:        float64(notification.TotalFee) / 100, // Convert from cents
		PaidAt:        paidAt,
		PaymentData:   paymentData,
	}, nil
}

// Refund processes a refund for WeChat Pay
func (wp *WechatProvider) Refund(orderNo string, amount float64) (*RefundResponse, error) {
	// WeChat Pay refund implementation would go here
	// This is a simplified placeholder
	return &RefundResponse{
		Success: false,
		Message: "WeChat Pay refund not implemented yet",
	}, errors.New("refund not implemented")
}

// Helper methods

func (wp *WechatProvider) generateNonceStr() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (wp *WechatProvider) generateSign(req WechatUnifiedOrderRequest) string {
	// Build parameter map
	params := map[string]string{
		"appid":            req.AppID,
		"mch_id":           req.MchID,
		"nonce_str":        req.NonceStr,
		"body":             req.Body,
		"out_trade_no":     req.OutTradeNo,
		"total_fee":        fmt.Sprintf("%d", req.TotalFee),
		"spbill_create_ip": req.SpbillCreateIP,
		"notify_url":       req.NotifyURL,
		"trade_type":       req.TradeType,
	}

	return wp.signParams(params)
}

func (wp *WechatProvider) verifyNotificationSign(notification WechatNotification) bool {
	// Build parameter map
	params := map[string]string{
		"return_code":    notification.ReturnCode,
		"return_msg":     notification.ReturnMsg,
		"appid":          notification.AppID,
		"mch_id":         notification.MchID,
		"nonce_str":      notification.NonceStr,
		"result_code":    notification.ResultCode,
		"openid":         notification.OpenID,
		"trade_type":     notification.TradeType,
		"bank_type":      notification.BankType,
		"total_fee":      fmt.Sprintf("%d", notification.TotalFee),
		"transaction_id": notification.TransactionID,
		"out_trade_no":   notification.OutTradeNo,
		"time_end":       notification.TimeEnd,
	}

	expectedSign := wp.signParams(params)
	return expectedSign == notification.Sign
}

func (wp *WechatProvider) signParams(params map[string]string) string {
	// Sort parameters
	keys := make([]string, 0, len(params))
	for key := range params {
		if params[key] != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	// Build query string
	var query []string
	for _, key := range keys {
		query = append(query, fmt.Sprintf("%s=%s", key, params[key]))
	}
	queryString := strings.Join(query, "&")

	// Add API key
	queryString += "&key=" + wp.APIKey

	// Generate MD5 hash
	hash := md5.Sum([]byte(queryString))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}