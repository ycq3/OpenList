package op

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils/random"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// CreateUserRegistration 创建用户注册申请
func CreateUserRegistration(email, username, password string) (*model.UserRegistration, error) {
	// 检查邮箱是否已存在
	if _, err := db.GetUserByName(email); err == nil {
		return nil, errors.New("邮箱已被注册")
	}
	
	// 检查用户名是否已存在
	if _, err := db.GetUserByName(username); err == nil {
		return nil, errors.New("用户名已被使用")
	}
	
	// 检查是否已有待处理的注册申请
	if existing, err := db.GetUserRegistrationByEmail(email); err == nil && !existing.IsExpired() {
		return nil, errors.New("已有待处理的注册申请，请稍后再试")
	}
	
	// 生成密码哈希和盐值
	salt := random.String(8)
	pwdHash := model.TwoHashPwd(password, salt)
	
	// 生成验证令牌
	token, err := generateToken(32)
	if err != nil {
		return nil, errors.Wrap(err, "生成验证令牌失败")
	}
	
	registration := &model.UserRegistration{
		Email:     email,
		Username:  username,
		Password:  password, // 临时存储明文密码用于验证
		PwdHash:   pwdHash,
		Salt:      salt,
		Status:    0, // 待验证
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24小时过期
	}
	
	err = db.CreateUserRegistration(registration)
	if err != nil {
		return nil, errors.Wrap(err, "创建注册申请失败")
	}
	
	return registration, nil
}

// VerifyUserRegistration 验证用户注册
func VerifyUserRegistration(token string) (*model.UserRegistration, error) {
	registration, err := db.GetUserRegistrationByToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无效的验证链接")
		}
		return nil, errors.Wrap(err, "获取注册信息失败")
	}
	
	if registration.IsExpired() {
		return nil, errors.New("验证链接已过期")
	}
	
	// 更新状态为已验证
	registration.Status = 1
	err = db.UpdateUserRegistration(registration)
	if err != nil {
		return nil, errors.Wrap(err, "更新注册状态失败")
	}
	
	return registration, nil
}

// ApproveUserRegistration 批准用户注册
func ApproveUserRegistration(registrationID uint) (*model.User, error) {
	registration, err := db.GetUserRegistrationByToken("")
	if err != nil {
		return nil, errors.Wrap(err, "获取注册信息失败")
	}
	
	if registration.Status != 1 {
		return nil, errors.New("注册申请未验证或已处理")
	}
	
	// 创建用户
	user := &model.User{
		Username:   registration.Username,
		PwdHash:    registration.PwdHash,
		Salt:       registration.Salt,
		BasePath:   "/",
		Role:       model.GENERAL, // 普通用户
		Disabled:   false,
		Permission: 0, // 默认权限
	}
	
	err = CreateUser(user)
	if err != nil {
		return nil, errors.Wrap(err, "创建用户失败")
	}
	
	// 创建用户积分账户
	credits := &model.UserCredits{
		UserID:  user.ID,
		Balance: 0, // 初始积分为0
	}
	err = db.CreateUserCredits(credits)
	if err != nil {
		return nil, errors.Wrap(err, "创建积分账户失败")
	}
	
	// 更新注册状态为已注册
	registration.Status = 2
	err = db.UpdateUserRegistration(registration)
	if err != nil {
		return nil, errors.Wrap(err, "更新注册状态失败")
	}
	
	return user, nil
}

// RejectUserRegistration 拒绝用户注册
func RejectUserRegistration(registrationID uint) error {
	registration, err := db.GetUserRegistrationByToken("")
	if err != nil {
		return errors.Wrap(err, "获取注册信息失败")
	}
	
	registration.Status = -1 // 已拒绝
	err = db.UpdateUserRegistration(registration)
	if err != nil {
		return errors.Wrap(err, "更新注册状态失败")
	}
	
	return nil
}

// CreateVerificationCode 创建验证码
func CreateVerificationCode(email, codeType string) (*model.VerificationCode, error) {
	// 生成6位数字验证码
	code := random.String(6)
	
	verificationCode := &model.VerificationCode{
		Email:     email,
		Code:      code,
		Type:      codeType,
		Used:      false,
		ExpiresAt: time.Now().Add(10 * time.Minute), // 10分钟过期
	}
	
	err := db.CreateVerificationCode(verificationCode)
	if err != nil {
		return nil, errors.Wrap(err, "创建验证码失败")
	}
	
	return verificationCode, nil
}

// VerifyCode 验证验证码
func VerifyCode(email, code, codeType string) error {
	verificationCode, err := db.GetVerificationCode(email, codeType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("验证码不存在或已过期")
		}
		return errors.Wrap(err, "获取验证码失败")
	}
	
	if !verificationCode.CanUse() {
		return errors.New("验证码已使用或已过期")
	}
	
	if verificationCode.Code != code {
		return errors.New("验证码错误")
	}
	
	// 标记为已使用
	verificationCode.Used = true
	err = db.UpdateVerificationCode(verificationCode)
	if err != nil {
		return errors.Wrap(err, "更新验证码状态失败")
	}
	
	return nil
}

// GetPendingRegistrations 获取待处理的注册申请
func GetPendingRegistrations(page, pageSize int) ([]model.UserRegistration, int64, error) {
	return db.GetPendingRegistrations(page, pageSize)
}

// CleanExpiredData 清理过期数据
func CleanExpiredData() error {
	// 清理过期的注册记录
	if err := db.CleanExpiredUserRegistrations(); err != nil {
		return errors.Wrap(err, "清理过期注册记录失败")
	}
	
	// 清理过期的验证码
	if err := db.CleanExpiredVerificationCodes(); err != nil {
		return errors.Wrap(err, "清理过期验证码失败")
	}
	
	return nil
}

// generateToken 生成随机令牌
func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SendVerificationEmail 发送验证邮件（占位函数，需要实现邮件发送逻辑）
func SendVerificationEmail(email, token string) error {
	// TODO: 实现邮件发送逻辑
	verifyURL := fmt.Sprintf("http://localhost:5244/api/auth/verify?token=%s", token)
	utils.Log.Infof("发送验证邮件到 %s，验证链接: %s", email, verifyURL)
	return nil
}

// SendVerificationCode 发送验证码（占位函数，需要实现邮件发送逻辑）
func SendVerificationCode(email, code string) error {
	// TODO: 实现邮件发送逻辑
	utils.Log.Infof("发送验证码到 %s，验证码: %s", email, code)
	return nil
}