package op

import (
	"fmt"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils/random"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// CreateUserCredits 创建用户积分账户
func CreateUserCredits(userID uint) (*model.UserCredits, error) {
	// 检查是否已存在积分账户
	if existing, err := db.GetUserCreditsByUserID(userID); err == nil {
		return existing, nil
	}

	credits := &model.UserCredits{
		UserID:  userID,
		Balance: 0,
	}

	err := db.CreateUserCredits(credits)
	if err != nil {
		return nil, errors.Wrap(err, "创建用户积分账户失败")
	}

	return credits, nil
}

// GetUserCredits 获取用户积分
func GetUserCredits(userID uint) (*model.UserCredits, error) {
	credits, err := db.GetUserCreditsByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果不存在，自动创建
			return CreateUserCredits(userID)
		}
		return nil, errors.Wrap(err, "获取用户积分失败")
	}
	return credits, nil
}

// AddCredits 增加用户积分
func AddCredits(userID uint, amount int64, reason, orderID string) error {
	credits, err := GetUserCredits(userID)
	if err != nil {
		return err
	}

	// 更新积分
	credits.Balance += amount
	credits.TotalEarn += amount
	err = db.UpdateUserCredits(credits)
	if err != nil {
		return errors.Wrap(err, "更新用户积分失败")
	}

	// 记录交易
	transaction := &model.CreditTransaction{
		UserID:      userID,
		Amount:      amount,
		Type:        "earn",
		Source:      reason,
		SourceID:    orderID,
		Balance:     credits.Balance,
		Description: reason,
	}

	err = db.CreateCreditTransaction(transaction)
	if err != nil {
		return errors.Wrap(err, "记录积分交易失败")
	}

	return nil
}

// DeductCredits 扣除用户积分
func DeductCredits(userID uint, amount int64, reason, fileID string) error {
	credits, err := GetUserCredits(userID)
	if err != nil {
		return err
	}

	if credits.Balance < amount {
		return errors.New("积分不足")
	}

	// 更新积分
	credits.Balance -= amount
	credits.TotalSpent += amount
	err = db.UpdateUserCredits(credits)
	if err != nil {
		return errors.Wrap(err, "更新用户积分失败")
	}

	// 记录交易
	transaction := &model.CreditTransaction{
		UserID:      userID,
		Amount:      -amount,
		Type:        "spend",
		Source:      "download",
		SourceID:    fileID,
		Balance:     credits.Balance,
		Description: reason,
	}

	err = db.CreateCreditTransaction(transaction)
	if err != nil {
		return errors.Wrap(err, "记录积分交易失败")
	}

	return nil
}

// GetCreditTransactions 获取用户积分交易记录
func GetCreditTransactions(userID uint, page, pageSize int) ([]model.CreditTransaction, int64, error) {
	return db.GetCreditTransactionsByUserID(userID, page, pageSize)
}

// SetFileCreditsConfig 设置文件积分配置
func SetFileCreditsConfig(path string, credits int64, isFolder bool, createdBy uint) error {
	config := &model.FileCreditsConfig{
		Path:      path,
		Credits:   credits,
		IsFolder:  isFolder,
		CreatedBy: createdBy,
	}

	err := db.CreateFileCreditsConfig(config)
	if err != nil {
		return errors.Wrap(err, "设置文件积分配置失败")
	}

	return nil
}

// GetFileCreditsConfig 获取文件积分配置
func GetFileCreditsConfig(path string) (*model.FileCreditsConfig, error) {
	config, err := db.GetFileCreditsConfigByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果没有找到配置，尝试继承父目录配置
			return db.GetInheritableCreditsConfig(path)
		}
		return nil, errors.Wrap(err, "获取文件积分配置失败")
	}
	return config, nil
}

// DeleteFileCreditsConfig 删除文件积分配置
func DeleteFileCreditsConfig(configID uint) error {
	err := db.DeleteFileCreditsConfig(configID)
	if err != nil {
		return errors.Wrap(err, "删除文件积分配置失败")
	}
	return nil
}

// GenerateRedeemCodes 批量生成兑换码
func GenerateRedeemCodes(count int, credits int64, description string, createdBy uint, expiresAt *time.Time) ([]string, error) {
	codes := make([]string, 0, count)

	for i := 0; i < count; i++ {
		code := generateRedeemCode()
		codes = append(codes, code)

		redeemCode := &model.RedeemCode{
			Code:        code,
			Credits:     credits,
			Description: description,
			CreatedBy:   createdBy,
			ExpiresAt:   expiresAt,
		}

		err := db.CreateRedeemCode(redeemCode)
		if err != nil {
			return nil, errors.Wrap(err, "创建兑换码失败")
		}
	}

	return codes, nil
}

// RedeemCode 兑换积分码
func RedeemCode(userID uint, code string) error {
	redeemCode, err := db.GetRedeemCodeByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("兑换码不存在")
		}
		return errors.Wrap(err, "获取兑换码失败")
	}

	if !redeemCode.CanUse() {
		return errors.New("兑换码已使用或已过期")
	}

	// 更新兑换码使用次数
	redeemCode.UsedCount++
	err = db.UpdateRedeemCode(redeemCode)
	if err != nil {
		return errors.Wrap(err, "更新兑换码状态失败")
	}

	// 记录使用记录
	usage := &model.RedeemCodeUsage{
		UserID:       userID,
		RedeemCodeID: redeemCode.ID,
		Credits:      redeemCode.Credits,
		UsedAt:       time.Now(),
	}
	err = db.CreateRedeemCodeUsage(usage)
	if err != nil {
		return errors.Wrap(err, "记录兑换码使用失败")
	}

	// 增加用户积分
	err = AddCredits(userID, redeemCode.Credits, fmt.Sprintf("兑换码: %s", code), "")
	if err != nil {
		return errors.Wrap(err, "增加积分失败")
	}

	return nil
}

// CreatePaymentOrder 创建支付订单
func CreatePaymentOrder(userID uint, amount int64, credits int64, paymentMethod string) (*model.PaymentOrder, error) {
	orderNo := generateOrderID()

	order := &model.PaymentOrder{
		OrderNo:       orderNo,
		UserID:        userID,
		Amount:        amount,
		Credits:       credits,
		PaymentMethod: paymentMethod,
		Status:        "pending",
		ExpiresAt:     time.Now().Add(30 * time.Minute), // 30分钟过期
	}

	err := db.CreatePaymentOrder(order)
	if err != nil {
		return nil, errors.Wrap(err, "创建支付订单失败")
	}

	return order, nil
}

// GetPaymentOrderByNo 根据订单号获取支付订单
func GetPaymentOrderByNo(orderNo string) (*model.PaymentOrder, error) {
	return db.GetPaymentOrderByOrderNo(orderNo)
}

// UpdatePaymentOrder 更新支付订单
func UpdatePaymentOrder(order *model.PaymentOrder) error {
	return db.UpdatePaymentOrder(order)
}

// ListPaymentOrders 获取用户支付订单列表
func ListPaymentOrders(userID uint, page, pageSize int) ([]model.PaymentOrder, int64, error) {
	return db.GetPaymentOrdersByUserID(userID, page, pageSize)
}

// CompletePaymentOrder 完成支付订单
func CompletePaymentOrder(orderNo string, transactionID string, amount float64, paidAt time.Time) error {
	order, err := db.GetPaymentOrderByOrderNo(orderNo)
	if err != nil {
		return errors.Wrap(err, "获取支付订单失败")
	}

	if order.Status != "pending" {
		return errors.New("订单状态异常")
	}

	if order.IsExpired() {
		return errors.New("订单已过期")
	}

	// 更新订单状态
	order.Status = "completed"
	order.PaymentData = fmt.Sprintf(`{"transaction_id":"%s"}`, transactionID)
	order.PaidAt = &paidAt

	err = db.UpdatePaymentOrder(order)
	if err != nil {
		return errors.Wrap(err, "更新支付订单失败")
	}

	// 增加用户积分
	err = AddCredits(order.UserID, order.Credits, fmt.Sprintf("购买积分: %s", orderNo), orderNo)
	if err != nil {
		return errors.Wrap(err, "增加积分失败")
	}

	return nil
}

// CancelPaymentOrder 取消支付订单
func CancelPaymentOrder(orderNo string, userID uint) error {
	order, err := db.GetPaymentOrderByOrderNo(orderNo)
	if err != nil {
		return errors.Wrap(err, "获取支付订单失败")
	}

	if order.Status != "pending" {
		return errors.New("订单状态异常")
	}

	order.Status = "cancelled"
	err = db.UpdatePaymentOrder(order)
	if err != nil {
		return errors.Wrap(err, "更新支付订单失败")
	}

	return nil
}

// CleanExpiredPaymentOrders 清理过期的支付订单
func CleanExpiredPaymentOrders() error {
	return db.CleanExpiredPaymentOrders()
}

// generateRedeemCode 生成兑换码
func generateRedeemCode() string {
	return "OL" + random.String(12)
}

// generateOrderID 生成订单ID
func generateOrderID() string {
	return fmt.Sprintf("OL%d%s", time.Now().Unix(), random.String(8))
}

// CheckFileDownloadPermission 检查文件下载权限和积分
func CheckFileDownloadPermission(userID uint, filePath string) (bool, int64, error) {
	// 获取文件积分配置
	config, err := GetFileCreditsConfig(filePath)
	if err != nil {
		// 如果没有配置，默认免费
		return true, 0, nil
	}

	if config.Credits <= 0 {
		// 免费文件
		return true, 0, nil
	}

	// 检查用户积分
	userCredits, err := GetUserCredits(userID)
	if err != nil {
		return false, config.Credits, err
	}

	if userCredits.Balance < config.Credits {
		return false, config.Credits, nil
	}

	return true, config.Credits, nil
}

// ProcessFileDownload 处理文件下载（扣除积分）
func ProcessFileDownload(userID uint, filePath string) error {
	canDownload, requiredCredits, err := CheckFileDownloadPermission(userID, filePath)
	if err != nil {
		return err
	}

	if !canDownload {
		return errors.New("积分不足")
	}

	if requiredCredits > 0 {
		err = DeductCredits(userID, requiredCredits, fmt.Sprintf("下载文件: %s", filePath), filePath)
		if err != nil {
			return err
		}
	}

	return nil
}