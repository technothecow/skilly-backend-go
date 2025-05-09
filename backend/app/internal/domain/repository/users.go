package repository

import (
	"context"
	"errors"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"skilly/internal/domain/models"
	imongo "skilly/internal/infrastructure/mongo"
)

type UserRepository interface {
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	CreateUser(ctx context.Context, user models.User) error
	UpdateUser(ctx context.Context, user models.User) error
	DeleteUser(ctx context.Context, user models.User) error
}

type UserRepositoryImpl struct {
	mongo *imongo.Client
	logger *slog.Logger
}

func NewUserRepository(m *imongo.Client, l *slog.Logger) *UserRepositoryImpl {
	return &UserRepositoryImpl{mongo: m, logger: l}
}

var ErrUserNotFound = errors.New("user not found")
var ErrInternal = errors.New("internal error")

func (r *UserRepositoryImpl) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	result := r.mongo.Database.Collection("users").FindOne(ctx, bson.M{"username": username})
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		r.logger.Error("failed to find user", slog.Any("error", result.Err()))
		return nil, ErrInternal
	}

	var user models.User
	err := result.Decode(&user)
	if err != nil {
		r.logger.Error("failed to decode user", slog.Any("error", err))
		return nil, ErrInternal
	}

	return &user, nil
}

func (r *UserRepositoryImpl) CreateUser(ctx context.Context, user models.User) error {
	_, err := r.mongo.Database.Collection("users").InsertOne(ctx, user)
	if err != nil {
		r.logger.Error("failed to create user", slog.Any("error", err))
		return ErrInternal
	}

	return nil
}

func (r *UserRepositoryImpl) UpdateUser(ctx context.Context, user models.User) error {
	_, err := r.mongo.Database.Collection("users").UpdateOne(ctx, bson.M{"_id": user.Id}, bson.M{"$set": user})
	if err != nil {
		r.logger.Error("failed to update user", slog.Any("error", err))
		return ErrInternal
	}

	return nil
}

func (r *UserRepositoryImpl) DeleteUser(ctx context.Context, user models.User) error {
	_, err := r.mongo.Database.Collection("users").DeleteOne(ctx, bson.M{"_id": user.Id})
	if err != nil {
		r.logger.Error("failed to delete user", slog.Any("error", err))
		return ErrInternal
	}

	return nil
}
