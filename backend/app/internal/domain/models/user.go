package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	Id         primitive.ObjectID `bson:"_id,omitempty"`
	Username   string             `json:"username"`
	Password   string             `json:"password"`
	Email      string             `json:"email"`
	Bio        string             `json:"bio"`
	Teaching   []string           `json:"teaching"`
	Learning   []string           `json:"learning"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}
