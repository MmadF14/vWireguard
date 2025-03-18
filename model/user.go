package model

// User represents a system user
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"` // This should be hashed in production
	Role     Role   `json:"role"`
}

// Role represents a user's role in the system
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
)
