// Package models – структуры данных, которые ходят между слоями и в JSON.
package models

// Роли пользователей.
const (
	RoleIntern = "intern"
	RoleMentor = "mentor"
	RoleLead   = "lead"
)

// User – публичный объект пользователя (контракт API, поля camelCase).
type User struct {
	ID       string  `json:"id"`
	Login    string  `json:"login"`
	Name     string  `json:"name"`
	Role     string  `json:"role"`
	TeamID   string  `json:"teamId"`
	TeamName string  `json:"teamName"`
	MentorID *string `json:"mentorId"`
}

// UserWithPassword – пользователь вместе с хешем пароля. Наружу никогда не отдается.
type UserWithPassword struct {
	User
	PasswordHash string
}

// TestUser – карточка тестового пользователя для экрана входа (без пароля).
type TestUser struct {
	Login    string `json:"login"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	TeamName string `json:"teamName"`
}
