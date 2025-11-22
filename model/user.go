package model

import (
	"time"
)

// UserRole تعریف نوع نقش کاربر
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleUser    UserRole = "user"
)

// User model
type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         UserRole  `json:"role"`
	APIToken     string    `json:"api_token,omitempty"`
	TokenExpire  time.Time `json:"token_expire,omitempty"`
}
