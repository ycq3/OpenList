package db

import (
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
)

// CreateUserRegistration 创建用户注册记录
func CreateUserRegistration(registration *model.UserRegistration) error {
	return db.Create(registration).Error
}

// GetUserRegistrationByToken 根据令牌获取注册记录
func GetUserRegistrationByToken(token string) (*model.UserRegistration, error) {
	var registration model.UserRegistration
	err := db.Where("token = ? AND status = 0", token).First(&registration).Error
	return &registration, err
}

// GetUserRegistrationByEmail 根据邮箱获取注册记录
func GetUserRegistrationByEmail(email string) (*model.UserRegistration, error) {
	var registration model.UserRegistration
	err := db.Where("email = ?", email).First(&registration).Error
	return &registration, err
}

// GetUserRegistrationByUsername 根据用户名获取注册记录
func GetUserRegistrationByUsername(username string) (*model.UserRegistration, error) {
	var registration model.UserRegistration
	err := db.Where("username = ?", username).First(&registration).Error
	return &registration, err
}

// UpdateUserRegistration 更新用户注册记录
func UpdateUserRegistration(registration *model.UserRegistration) error {
	return db.Save(registration).Error
}

// DeleteUserRegistration 删除用户注册记录
func DeleteUserRegistration(id uint) error {
	return db.Delete(&model.UserRegistration{}, id).Error
}

// CleanExpiredUserRegistrations 清理过期的注册记录
func CleanExpiredUserRegistrations() error {
	return db.Where("expires_at < ? AND status = 0", time.Now()).Delete(&model.UserRegistration{}).Error
}

// CreateVerificationCode 创建验证码记录
func CreateVerificationCode(code *model.VerificationCode) error {
	return db.Create(code).Error
}

// GetVerificationCode 获取验证码记录
func GetVerificationCode(email, codeType string) (*model.VerificationCode, error) {
	var code model.VerificationCode
	err := db.Where("email = ? AND type = ? AND used = false AND expires_at > ?", 
		email, codeType, time.Now()).Order("created_at DESC").First(&code).Error
	return &code, err
}

// UpdateVerificationCode 更新验证码记录
func UpdateVerificationCode(code *model.VerificationCode) error {
	return db.Save(code).Error
}

// CleanExpiredVerificationCodes 清理过期的验证码
func CleanExpiredVerificationCodes() error {
	return db.Where("expires_at < ?", time.Now()).Delete(&model.VerificationCode{}).Error
}

// GetPendingRegistrations 获取待处理的注册申请
func GetPendingRegistrations(page, pageSize int) ([]model.UserRegistration, int64, error) {
	var registrations []model.UserRegistration
	var total int64
	
	query := db.Model(&model.UserRegistration{}).Where("status = 0")
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&registrations).Error
	return registrations, total, err
}