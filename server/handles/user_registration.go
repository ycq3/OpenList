package handles

import (
	"strconv"

	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

// CreateRegistrationReq 创建用户注册申请请求
type CreateRegistrationReq struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Reason   string `json:"reason" binding:"max=500"` // 申请理由
}

// CreateRegistration 创建用户注册申请
func CreateRegistration(c *gin.Context) {
	var req CreateRegistrationReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// 创建注册申请
	registration, err := op.CreateUserRegistration(req.Username, req.Email, req.Password)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"id":      registration.ID,
		"message": "Registration application submitted successfully. Please wait for admin approval.",
	})
}

// VerifyRegistrationReq 验证注册申请请求
type VerifyRegistrationReq struct {
	Token string `json:"token" binding:"required"`
}

// VerifyRegistration 验证用户注册申请
func VerifyRegistration(c *gin.Context) {
	var req VerifyRegistrationReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// 验证注册申请
	_, err := op.VerifyUserRegistration(req.Token)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "Registration verified successfully.",
	})
}

// ApproveRegistrationReq 批准注册申请请求
type ApproveRegistrationReq struct {
	ID uint `json:"id" binding:"required"`
}

// ApproveRegistration 批准用户注册申请（管理员）
func ApproveRegistration(c *gin.Context) {
	var req ApproveRegistrationReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// 批准注册申请
	user, err := op.ApproveUserRegistration(req.ID)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"user_id": user.ID,
		"message": "Registration approved successfully.",
	})
}

// RejectRegistrationReq 拒绝注册申请请求
type RejectRegistrationReq struct {
	ID     uint   `json:"id" binding:"required"`
	Reason string `json:"reason" binding:"max=500"` // 拒绝理由
}

// RejectRegistration 拒绝用户注册申请（管理员）
func RejectRegistration(c *gin.Context) {
	var req RejectRegistrationReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// 拒绝注册申请
	err := op.RejectUserRegistration(req.ID)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "Registration rejected successfully.",
	})
}

// ListPendingRegistrations 获取待处理的注册申请列表（管理员）
func ListPendingRegistrations(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取待处理的注册申请
	registrations, total, err := op.GetPendingRegistrations(page, pageSize)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 500)
		return
	}

	common.SuccessResp(c, gin.H{
		"registrations": registrations,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

// SendVerificationCodeReq 发送验证码请求
type SendVerificationCodeReq struct {
	Email string `json:"email" binding:"required,email"`
	Type  string `json:"type" binding:"required,oneof=email sms"` // 验证码类型
}

// SendVerificationCode 发送验证码
func SendVerificationCode(c *gin.Context) {
	var req SendVerificationCodeReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// 创建验证码
	code, err := op.CreateVerificationCode(req.Email, req.Type)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message":   "Verification code sent successfully.",
		"code_id":   code.ID,
		"expires_at": code.ExpiresAt,
	})
}

// VerifyCodeReq 验证验证码请求
type VerifyCodeReq struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
	Type  string `json:"type" binding:"required,oneof=email sms"`
}

// VerifyCode 验证验证码
func VerifyCode(c *gin.Context) {
	var req VerifyCodeReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// 验证验证码
	err := op.VerifyCode(req.Email, req.Code, req.Type)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	common.SuccessResp(c, gin.H{
		"message": "Verification code is valid.",
	})
}