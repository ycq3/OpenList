package payment

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/pkg/errors"
)

// PaymentProvider defines the interface for payment providers
type PaymentProvider interface {
	CreateOrder(order *model.PaymentOrder) (*PaymentResponse, error)
	VerifyPayment(orderNo string, paymentData map[string]interface{}) (*PaymentVerification, error)
	Refund(orderNo string, amount float64) (*RefundResponse, error)
}

// PaymentResponse represents the response from payment provider
type PaymentResponse struct {
	OrderNo     string                 `json:"order_no"`
	PaymentURL  string                 `json:"payment_url,omitempty"`
	QRCode      string                 `json:"qr_code,omitempty"`
	PaymentData map[string]interface{} `json:"payment_data"`
}

// PaymentVerification represents payment verification result
type PaymentVerification struct {
	Success       bool                   `json:"success"`
	OrderNo       string                 `json:"order_no"`
	TransactionID string                 `json:"transaction_id"`
	Amount        float64                `json:"amount"`
	PaidAt        time.Time              `json:"paid_at"`
	PaymentData   map[string]interface{} `json:"payment_data"`
}

// RefundResponse represents refund operation result
type RefundResponse struct {
	Success   bool   `json:"success"`
	RefundID  string `json:"refund_id"`
	Message   string `json:"message"`
}

// PaymentManager manages different payment providers
type PaymentManager struct {
	providers map[string]PaymentProvider
}

// NewPaymentManager creates a new payment manager
func NewPaymentManager() *PaymentManager {
	return &PaymentManager{
		providers: make(map[string]PaymentProvider),
	}
}

// RegisterProvider registers a payment provider
func (pm *PaymentManager) RegisterProvider(name string, provider PaymentProvider) {
	pm.providers[name] = provider
}

// GetProvider gets a payment provider by name
func (pm *PaymentManager) GetProvider(name string) (PaymentProvider, error) {
	provider, exists := pm.providers[name]
	if !exists {
		return nil, errors.Errorf("payment provider %s not found", name)
	}
	return provider, nil
}

// CreatePayment creates a payment order using specified provider
func (pm *PaymentManager) CreatePayment(order *model.PaymentOrder) (*PaymentResponse, error) {
	provider, err := pm.GetProvider(order.PaymentMethod)
	if err != nil {
		return nil, err
	}
	return provider.CreateOrder(order)
}

// VerifyPayment verifies a payment using specified provider
func (pm *PaymentManager) VerifyPayment(providerName, orderNo string, paymentData map[string]interface{}) (*PaymentVerification, error) {
	provider, err := pm.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	return provider.VerifyPayment(orderNo, paymentData)
}

// ProcessRefund processes a refund using specified provider
func (pm *PaymentManager) ProcessRefund(providerName, orderNo string, amount float64) (*RefundResponse, error) {
	provider, err := pm.GetProvider(providerName)
	if err != nil {
		return nil, err
	}
	return provider.Refund(orderNo, amount)
}

// Global payment manager instance
var DefaultPaymentManager = NewPaymentManager()

// Helper functions for JSON marshaling/unmarshaling payment data
func MarshalPaymentData(data map[string]interface{}) (string, error) {
	if data == nil {
		return "{}", nil
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal payment data")
	}
	return string(bytes), nil
}

func UnmarshalPaymentData(data string) (map[string]interface{}, error) {
	if data == "" {
		return make(map[string]interface{}), nil
	}
	var result map[string]interface{}
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal payment data")
	}
	return result, nil
}

// GenerateOrderNo generates a unique order number
func GenerateOrderNo() string {
	return fmt.Sprintf("OL%d", time.Now().UnixNano())
}

// Global payment manager instance
var globalPaymentManager *PaymentManager

// InitPaymentManager initializes the global payment manager
func InitPaymentManager() {
	globalPaymentManager = NewPaymentManager()
	
	// Register payment providers here
	// Example:
	// alipayConfig := AlipayConfig{...}
	// alipayProvider, _ := NewAlipayProvider(alipayConfig)
	// globalPaymentManager.RegisterProvider("alipay", alipayProvider)
	
	// wechatConfig := WechatConfig{...}
	// wechatProvider := NewWechatProvider(wechatConfig)
	// globalPaymentManager.RegisterProvider("wechat", wechatProvider)
}

// GetPaymentManager returns the global payment manager instance
func GetPaymentManager() *PaymentManager {
	if globalPaymentManager == nil {
		InitPaymentManager()
	}
	return globalPaymentManager
}