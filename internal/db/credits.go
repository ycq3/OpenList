package db

import (
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
)

// CreateUserCredits 创建用户积分账户
func CreateUserCredits(credits *model.UserCredits) error {
	return db.Create(credits).Error
}

// GetUserCreditsByUserID 根据用户ID获取积分账户
func GetUserCreditsByUserID(userID uint) (*model.UserCredits, error) {
	var credits model.UserCredits
	err := db.Where("user_id = ?", userID).First(&credits).Error
	return &credits, err
}

// UpdateUserCredits 更新用户积分账户
func UpdateUserCredits(credits *model.UserCredits) error {
	return db.Save(credits).Error
}

// CreateCreditTransaction 创建积分交易记录
func CreateCreditTransaction(transaction *model.CreditTransaction) error {
	return db.Create(transaction).Error
}

// GetCreditTransactionsByUserID 获取用户积分交易记录
func GetCreditTransactionsByUserID(userID uint, page, pageSize int) ([]model.CreditTransaction, int64, error) {
	var transactions []model.CreditTransaction
	var total int64
	
	query := db.Model(&model.CreditTransaction{}).Where("user_id = ?", userID)
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&transactions).Error
	return transactions, total, err
}

// CreateFileCreditsConfig 创建文件积分配置
func CreateFileCreditsConfig(config *model.FileCreditsConfig) error {
	return db.Create(config).Error
}

// GetFileCreditsConfigByPath 根据路径获取积分配置
func GetFileCreditsConfigByPath(path string) (*model.FileCreditsConfig, error) {
	var config model.FileCreditsConfig
	err := db.Where("path = ? AND enabled = true", path).First(&config).Error
	return &config, err
}

// GetFileCreditsConfigs 获取文件积分配置列表
func GetFileCreditsConfigs(page, pageSize int) ([]model.FileCreditsConfig, int64, error) {
	var configs []model.FileCreditsConfig
	var total int64
	
	query := db.Model(&model.FileCreditsConfig{})
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * pageSize
	err = query.Preload("Creator").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&configs).Error
	return configs, total, err
}

// UpdateFileCreditsConfig 更新文件积分配置
func UpdateFileCreditsConfig(config *model.FileCreditsConfig) error {
	return db.Save(config).Error
}

// DeleteFileCreditsConfig 删除文件积分配置
func DeleteFileCreditsConfig(id uint) error {
	return db.Delete(&model.FileCreditsConfig{}, id).Error
}

// GetInheritableCreditsConfig 获取可继承的积分配置
func GetInheritableCreditsConfig(path string) (*model.FileCreditsConfig, error) {
	var config model.FileCreditsConfig
	// 查找最匹配的父级配置
	err := db.Where("? LIKE CONCAT(path, '%') AND is_folder = true AND inheritable = true AND enabled = true", path).
		Order("LENGTH(path) DESC").First(&config).Error
	return &config, err
}

// CreateRedeemCode 创建兑换码
func CreateRedeemCode(code *model.RedeemCode) error {
	return db.Create(code).Error
}

// GetRedeemCodeByCode 根据兑换码获取记录
func GetRedeemCodeByCode(code string) (*model.RedeemCode, error) {
	var redeemCode model.RedeemCode
	err := db.Where("code = ? AND enabled = true", code).First(&redeemCode).Error
	return &redeemCode, err
}

// GetRedeemCodes 获取兑换码列表
func GetRedeemCodes(page, pageSize int) ([]model.RedeemCode, int64, error) {
	var codes []model.RedeemCode
	var total int64
	
	query := db.Model(&model.RedeemCode{})
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * pageSize
	err = query.Preload("Creator").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&codes).Error
	return codes, total, err
}

// UpdateRedeemCode 更新兑换码
func UpdateRedeemCode(code *model.RedeemCode) error {
	return db.Save(code).Error
}

// CreateRedeemCodeUsage 创建兑换码使用记录
func CreateRedeemCodeUsage(usage *model.RedeemCodeUsage) error {
	return db.Create(usage).Error
}

// GetRedeemCodeUsages 获取兑换码使用记录
func GetRedeemCodeUsages(redeemCodeID uint, page, pageSize int) ([]model.RedeemCodeUsage, int64, error) {
	var usages []model.RedeemCodeUsage
	var total int64
	
	query := db.Model(&model.RedeemCodeUsage{}).Where("redeem_code_id = ?", redeemCodeID)
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * pageSize
	err = query.Preload("User").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&usages).Error
	return usages, total, err
}

// CreatePaymentOrder 创建支付订单
func CreatePaymentOrder(order *model.PaymentOrder) error {
	return db.Create(order).Error
}

// GetPaymentOrderByOrderNo 根据订单号获取订单
func GetPaymentOrderByOrderNo(orderNo string) (*model.PaymentOrder, error) {
	var order model.PaymentOrder
	err := db.Where("order_no = ?", orderNo).First(&order).Error
	return &order, err
}

// GetPaymentOrdersByUserID 获取用户支付订单
func GetPaymentOrdersByUserID(userID uint, page, pageSize int) ([]model.PaymentOrder, int64, error) {
	var orders []model.PaymentOrder
	var total int64
	
	query := db.Model(&model.PaymentOrder{}).Where("user_id = ?", userID)
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error
	return orders, total, err
}

// UpdatePaymentOrder 更新支付订单
func UpdatePaymentOrder(order *model.PaymentOrder) error {
	return db.Save(order).Error
}

// CleanExpiredPaymentOrders 清理过期的支付订单
func CleanExpiredPaymentOrders() error {
	return db.Where("expires_at < ? AND status = 'pending'", time.Now()).Update("status", "expired").Error
}