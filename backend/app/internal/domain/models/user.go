package models

import "time"

type User struct {
	Id        int64     `json:"_id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Email     string    `json:"email"`
	Bio       string    `json:"bio"`
	Teaching  []string  `json:"teaching"`
	Learning  []string  `json:"learning"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}