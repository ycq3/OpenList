package model

import (
	"time"

	"gorm.io/gorm"
)

// UserRegistration 用户注册记录
type UserRegistration struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"not null"` // 明文密码，仅用于临时存储
	PwdHash   string         `json:"-" gorm:"not null"` // 密码哈希
	Salt      string         `json:"-" gorm:"not null"` // 密码盐值
	Status    int            `json:"status" gorm:"default:0"` // 0: 待验证, 1: 已验证, 2: 已注册, -1: 已拒绝
	Token     string         `json:"-" gorm:"uniqueIndex"` // 验证令牌
	ExpiresAt time.Time      `json:"expires_at"` // 令牌过期时间
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// VerificationCode 验证码记录
type VerificationCode struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Email     string         `json:"email" gorm:"index;not null"`
	Code      string         `json:"-" gorm:"not null"` // 验证码
	Type      string         `json:"type" gorm:"not null"` // 验证码类型: register, reset_password
	Used      bool           `json:"used" gorm:"default:false"` // 是否已使用
	ExpiresAt time.Time      `json:"expires_at"` // 过期时间
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 设置表名
func (UserRegistration) TableName() string {
	return "x_user_registrations"
}

// TableName 设置表名
func (VerificationCode) TableName() string {
	return "x_verification_codes"
}

// IsExpired 检查注册记录是否过期
func (ur *UserRegistration) IsExpired() bool {
	return time.Now().After(ur.ExpiresAt)
}

// IsExpired 检查验证码是否过期
func (vc *VerificationCode) IsExpired() bool {
	return time.Now().After(vc.ExpiresAt)
}

// CanUse 检查验证码是否可用
func (vc *VerificationCode) CanUse() bool {
	return !vc.Used && !vc.IsExpired()
}