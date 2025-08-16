package model

import (
	"time"

	"gorm.io/gorm"
)

// UserCredits 用户积分账户
type UserCredits struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"uniqueIndex;not null"` // 关联用户ID
	Balance   int64          `json:"balance" gorm:"default:0"` // 积分余额
	TotalEarn int64          `json:"total_earn" gorm:"default:0"` // 累计获得积分
	TotalSpent int64         `json:"total_spent" gorm:"default:0"` // 累计消费积分
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	User      *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// CreditTransaction 积分交易记录
type CreditTransaction struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"index;not null"` // 用户ID
	Type        string         `json:"type" gorm:"not null"` // 交易类型: earn, spend, refund
	Amount      int64          `json:"amount" gorm:"not null"` // 积分数量（正数为获得，负数为消费）
	Balance     int64          `json:"balance" gorm:"not null"` // 交易后余额
	Source      string         `json:"source" gorm:"not null"` // 来源: purchase, redeem_code, download, admin
	SourceID    string         `json:"source_id"` // 来源ID（如订单ID、兑换码ID等）
	Description string         `json:"description"` // 交易描述
	Metadata    string         `json:"metadata" gorm:"type:text"` // 额外元数据（JSON格式）
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	User        *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// FileCreditsConfig 文件积分配置
type FileCreditsConfig struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Path        string         `json:"path" gorm:"uniqueIndex;not null"` // 文件或文件夹路径
	IsFolder    bool           `json:"is_folder" gorm:"default:false"` // 是否为文件夹配置
	Credits     int64          `json:"credits" gorm:"not null"` // 所需积分
	Inheritable bool           `json:"inheritable" gorm:"default:true"` // 子文件是否继承此配置
	Enabled     bool           `json:"enabled" gorm:"default:true"` // 是否启用
	CreatedBy   uint           `json:"created_by" gorm:"not null"` // 创建者ID
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	Creator     *User          `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// RedeemCode 兑换码
type RedeemCode struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Code        string         `json:"code" gorm:"uniqueIndex;not null"` // 兑换码
	Credits     int64          `json:"credits" gorm:"not null"` // 积分数量
	MaxUses     int            `json:"max_uses" gorm:"default:1"` // 最大使用次数
	UsedCount   int            `json:"used_count" gorm:"default:0"` // 已使用次数
	Enabled     bool           `json:"enabled" gorm:"default:true"` // 是否启用
	ExpiresAt   *time.Time     `json:"expires_at"` // 过期时间（可为空）
	CreatedBy   uint           `json:"created_by" gorm:"not null"` // 创建者ID
	Description string         `json:"description"` // 描述
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	Creator     *User          `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// RedeemCodeUsage 兑换码使用记录
type RedeemCodeUsage struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	RedeemCodeID uint           `json:"redeem_code_id" gorm:"index;not null"` // 兑换码ID
	UserID       uint           `json:"user_id" gorm:"index;not null"` // 用户ID
	Credits      int64          `json:"credits" gorm:"not null"` // 获得的积分
	UsedAt       time.Time      `json:"used_at"` // 使用时间
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	RedeemCode   *RedeemCode    `json:"redeem_code,omitempty" gorm:"foreignKey:RedeemCodeID"`
	User         *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// PaymentOrder 支付订单
type PaymentOrder struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	OrderNo       string         `json:"order_no" gorm:"uniqueIndex;not null"` // 订单号
	UserID        uint           `json:"user_id" gorm:"index;not null"` // 用户ID
	Credits       int64          `json:"credits" gorm:"not null"` // 购买积分数量
	Amount        int64          `json:"amount" gorm:"not null"` // 支付金额（分）
	Currency      string         `json:"currency" gorm:"default:'CNY'"` // 货币类型
	PaymentMethod string         `json:"payment_method"` // 支付方式
	Status        string         `json:"status" gorm:"default:'pending'"` // 订单状态: pending, paid, failed, cancelled
	PaidAt        *time.Time     `json:"paid_at"` // 支付时间
	ExpiresAt     time.Time      `json:"expires_at"` // 订单过期时间
	PaymentData   string         `json:"payment_data" gorm:"type:text"` // 支付相关数据（JSON格式）
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	User          *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 设置表名
func (UserCredits) TableName() string {
	return "x_user_credits"
}

func (CreditTransaction) TableName() string {
	return "x_credit_transactions"
}

func (FileCreditsConfig) TableName() string {
	return "x_file_credits_configs"
}

func (RedeemCode) TableName() string {
	return "x_redeem_codes"
}

func (RedeemCodeUsage) TableName() string {
	return "x_redeem_code_usages"
}

func (PaymentOrder) TableName() string {
	return "x_payment_orders"
}

// IsExpired 检查兑换码是否过期
func (rc *RedeemCode) IsExpired() bool {
	if rc.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*rc.ExpiresAt)
}

// CanUse 检查兑换码是否可用
func (rc *RedeemCode) CanUse() bool {
	return rc.Enabled && !rc.IsExpired() && rc.UsedCount < rc.MaxUses
}

// IsExpired 检查支付订单是否过期
func (po *PaymentOrder) IsExpired() bool {
	return time.Now().After(po.ExpiresAt)
}

// IsPaid 检查订单是否已支付
func (po *PaymentOrder) IsPaid() bool {
	return po.Status == "paid"
}