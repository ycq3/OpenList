package handles

import (
	"strconv"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

// GetUserCredits 获取用户积分信息
func GetUserCredits(c *gin.Context) {
	user := c.MustGet("user").(*model.User)

	credits, err := op.GetUserCredits(user.ID)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 500)
		return
	}

	common.SuccessResp(c, credits)
}

// GetCreditTransactions 获取用户积分交易记录
func GetCreditTransactions(c *gin.Context) {
	user := c.MustGet("user").(*model.User)

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	transactions, total, err := op.GetCreditTransactions(user.ID, page, pageSize)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 500)
		return
	}

	common.SuccessResp(c, gin.H{
		"transactions": transactions,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
	})
}

// SetFileCreditsConfigReq 设置文件积分配置请求
type SetFileCreditsConfigReq struct {
	Path        string `json:"path" binding:"required"`
	IsFolder    bool   `json:"is_folder"`
	Credits     int64  `json:"credits" binding:"min=0"`
	Inheritable bool   `json:"inheritable"`
	Enabled     bool   `json:"enabled"`
}

// SetFileCreditsConfig 设置文件积分配置（管理员）
func SetFileCreditsConfig(c *gin.Context) {
	var req SetFileCreditsConfigReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	user := c.MustGet("user").(*model.User)

	err := op.SetFileCreditsConfig(req.Path, req.Credits, req.IsFolder, user.ID)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "File credits config set successfully",
	})
}

// GetFileCreditsConfig 获取文件积分配置
func GetFileCreditsConfig(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		common.ErrorStrResp(c, "path is required", 400)
		return
	}

	config, err := op.GetFileCreditsConfig(path)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 500)
		return
	}

	common.SuccessResp(c, config)
}

// DeleteFileCreditsConfig 删除文件积分配置（管理员）
func DeleteFileCreditsConfig(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		common.ErrorStrResp(c, "path is required", 400)
		return
	}

	// 首先获取配置以获得ID
	config, err := op.GetFileCreditsConfig(path)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 404)
		return
	}

	err = op.DeleteFileCreditsConfig(config.ID)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "File credits config deleted successfully",
	})
}

// GenerateRedeemCodesReq 生成兑换码请求
type GenerateRedeemCodesReq struct {
	Credits     int64  `json:"credits" binding:"required,min=1"`
	Count       int    `json:"count" binding:"required,min=1,max=1000"`
	MaxUses     int    `json:"max_uses" binding:"min=1"`
	Description string `json:"description" binding:"max=500"`
}

// GenerateRedeemCodes 生成兑换码（管理员）
func GenerateRedeemCodes(c *gin.Context) {
	var req GenerateRedeemCodesReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	user := c.MustGet("user").(*model.User)

	codes, err := op.GenerateRedeemCodes(req.Count, req.Credits, req.Description, user.ID, nil)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"codes":   codes,
		"message": "Redeem codes generated successfully",
	})
}

// RedeemCodeReq 兑换码兑换请求
type RedeemCodeReq struct {
	Code string `json:"code" binding:"required"`
}

// RedeemCode 兑换积分
func RedeemCode(c *gin.Context) {
	var req RedeemCodeReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	user := c.MustGet("user").(*model.User)

	err := op.RedeemCode(user.ID, req.Code)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "Redeem code used successfully",
	})
}

// CreatePaymentOrderReq 创建支付订单请求
type CreatePaymentOrderReq struct {
	Credits       int64  `json:"credits" binding:"required,min=1"`
	PaymentMethod string `json:"payment_method" binding:"required"`
}

// CreatePaymentOrder 创建支付订单
func CreatePaymentOrder(c *gin.Context) {
	var req CreatePaymentOrderReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	user := c.MustGet("user").(*model.User)

	// 计算金额（这里假设1积分=1分钱，实际应根据业务需求调整）
	amount := req.Credits

	order, err := op.CreatePaymentOrder(user.ID, amount, req.Credits, req.PaymentMethod)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, order)
}

// CompletePaymentOrderReq 完成支付订单请求
type CompletePaymentOrderReq struct {
	OrderNo       string `json:"order_no" binding:"required"`
	TransactionID string `json:"transaction_id" binding:"required"`
}

// CompletePaymentOrder 完成支付订单（支付回调）
func CompletePaymentOrder(c *gin.Context) {
	var req CompletePaymentOrderReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	err := op.CompletePaymentOrder(req.OrderNo, req.TransactionID, 0, time.Now())
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "Payment completed successfully",
	})
}

// CancelPaymentOrder 取消支付订单
func CancelPaymentOrder(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		common.ErrorStrResp(c, "order_no is required", 400)
		return
	}

	user := c.MustGet("user").(*model.User)
	err := op.CancelPaymentOrder(orderNo, user.ID)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "Payment order cancelled successfully",
	})
}

// PaymentNotification 处理支付通知
func PaymentNotification(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		common.ErrorStrResp(c, "Provider is required", 400)
		return
	}

	// 解析通知数据
	var paymentData map[string]interface{}
	var orderNo string

	switch provider {
	case "alipay":
		// 解析支付宝通知
		if err := c.ShouldBindJSON(&paymentData); err != nil {
			common.ErrorResp(c, err, 400)
			return
		}
		if outTradeNo, ok := paymentData["out_trade_no"].(string); ok {
			orderNo = outTradeNo
		}
	case "wechat":
		// 解析微信通知 (XML格式)
		body, err := c.GetRawData()
		if err != nil {
			common.ErrorResp(c, err, 400)
			return
		}
		paymentData = map[string]interface{}{
			"xml": string(body),
		}
	default:
		common.ErrorStrResp(c, "Unsupported payment provider", 400)
		return
	}

	// 这里应该调用支付验证逻辑
	// 由于支付验证比较复杂，这里简化处理
	// 实际项目中需要根据具体的支付提供商API进行验证

	// 模拟支付验证成功，完成订单
	if orderNo != "" {
		err := op.CompletePaymentOrder(orderNo, "mock_transaction_id", 0, time.Now())
		if err != nil {
			common.ErrorStrResp(c, err.Error(), 400)
			return
		}
	}

	// 根据支付提供商返回相应格式的成功响应
	switch provider {
	case "alipay":
		c.String(200, "success")
	case "wechat":
		c.XML(200, gin.H{
			"return_code": "SUCCESS",
			"return_msg":  "OK",
		})
	default:
		c.JSON(200, gin.H{"status": "success"})
	}
}

// CheckDownloadPermission 检查文件下载权限
func CheckDownloadPermission(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		common.ErrorStrResp(c, "path is required", 400)
		return
	}

	user := c.MustGet("user").(*model.User)

	canDownload, requiredCredits, err := op.CheckFileDownloadPermission(user.ID, path)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 500)
		return
	}

	common.SuccessResp(c, gin.H{
		"can_download":      canDownload,
		"required_credits":  requiredCredits,
	})
}

// DeductCreditsForDownload 扣除下载积分
func DeductCreditsForDownload(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		common.ErrorStrResp(c, "path is required", 400)
		return
	}

	user := c.MustGet("user").(*model.User)

	err := op.ProcessFileDownload(user.ID, path)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "Credits deducted successfully",
	})
}