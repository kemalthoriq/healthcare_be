package auth

import "time"

const (
	RoleAdmin     = "admin"
	RoleTherapist = "therapist"
)

type User struct {
	ID        int64
	Name      string
	Email     string
	Role      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewUser struct {
	Name         string
	Email        string
	PasswordHash string
	Role         string
}
