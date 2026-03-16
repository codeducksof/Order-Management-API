package domain

import (
	"time"

	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type User struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Order struct {
	ID            string           `json:"id" gorm:"type:uuid;primaryKey"`
	UserID        string           `json:"user_id" gorm:"type:uuid;not null;index"`
	CustomerName  string           `json:"customer_name" gorm:"not null"`
	TotalAmount   float64          `json:"total_amount" gorm:"not null"`
	Status        OrderStatus      `json:"status" gorm:"type:varchar(20);default:'pending'"`
	ExternalRef   string           `json:"external_ref,omitempty" gorm:"index"` // from mock external API
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	DeletedAt     gorm.DeletedAt   `json:"deleted_at,omitempty" gorm:"index"`
}

func (Order) TableName() string {
	return "orders"
}

func (User) TableName() string {
	return "users"
}
