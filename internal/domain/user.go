package domain

import (
	"context"
	"io"
	"time"
)

type Role string

const (
	RoleManager Role = "manager"
	RoleMember  Role = "member"
)

type User struct {
	ID        uint      `json:"userId" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"` // Hashed password, not sent in JSON
	Role      Role      `json:"role" gorm:"not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uint) (*User, error)
	GetAll(ctx context.Context) ([]*User, error)
}

type ImportResult struct {
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

type UserUseCase interface {
	Register(ctx context.Context, req *RegisterRequest) (*User, error)
	Login(ctx context.Context, req *LoginRequest) (string, error)
	GetUsers(ctx context.Context) ([]*User, error)
	ImportUsers(ctx context.Context, csvData io.Reader) (*ImportResult, error)
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
