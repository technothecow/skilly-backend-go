package repository

import (
	"context"
	"errors"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"skilly/internal/domain/models"
	imongo "skilly/internal/infrastructure/mongo"
)

type UserRepository interface {
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	CreateUser(ctx context.Context, user models.User) error
	UpdateUser(ctx context.Context, user models.User) error
	DeleteUser(ctx context.Context, user models.User) error
	SearchUsers(ctx context.Context, excludeUsername string, usernameSubstring string, learning []string, teaching []string, page int64, pagesize int64) ([]models.User, error)
}

type userRepositoryImpl struct {
	mongo  *imongo.Client
	logger *slog.Logger
}

func NewUserRepository(m *imongo.Client, l *slog.Logger) UserRepository {
	return &userRepositoryImpl{mongo: m, logger: l}
}

var ErrUserNotFound = errors.New("user not found")
var ErrInternal = errors.New("internal error")

const (
	usersCollectionName = "users"
)

func (r *userRepositoryImpl) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	result := r.mongo.Database.Collection(usersCollectionName).FindOne(ctx, bson.M{"username": username})
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

func (r *userRepositoryImpl) CreateUser(ctx context.Context, user models.User) error {
	_, err := r.mongo.Database.Collection(usersCollectionName).InsertOne(ctx, user)
	if err != nil {
		r.logger.Error("failed to create user", slog.Any("error", err))
		return ErrInternal
	}

	return nil
}

func (r *userRepositoryImpl) UpdateUser(ctx context.Context, user models.User) error {
	_, err := r.mongo.Database.Collection(usersCollectionName).UpdateOne(ctx, bson.M{"_id": user.Id}, bson.M{"$set": user})
	if err != nil {
		r.logger.Error("failed to update user", slog.Any("error", err))
		return ErrInternal
	}

	return nil
}

func (r *userRepositoryImpl) DeleteUser(ctx context.Context, user models.User) error {
	_, err := r.mongo.Database.Collection(usersCollectionName).DeleteOne(ctx, bson.M{"_id": user.Id})
	if err != nil {
		r.logger.Error("failed to delete user", slog.Any("error", err))
		return ErrInternal
	}

	return nil
}

/*
	SearchUsers searches for users by username that are teaching at least one of the given skills and learning at least one of the given skills.
*/
func (r *userRepositoryImpl) SearchUsers(ctx context.Context, excludeUsername string, usernameSubstring string, learning []string, teaching []string, page int64, pagesize int64) ([]models.User, error) {
	filter := bson.M{}

	if len(excludeUsername) > 0 {
		filter["username"] = bson.M{"$ne": excludeUsername}
	}

	if len(usernameSubstring) > 0 {
		filter["username"] = bson.M{"$regex": usernameSubstring, "$options": "i"}
	}

	if len(learning) > 0 {
		filter["learning"] = bson.M{"$in": learning}
	}

	if len(teaching) > 0 {
		filter["teaching"] = bson.M{"$in": teaching}
	}

	opts := options.Find().SetSkip(page * pagesize).SetLimit(pagesize)

	cur, err := r.mongo.Database.Collection(usersCollectionName).Find(ctx, filter, opts)
	if err != nil {
		// TODO: move logging to handlers
		r.logger.Error("failed to find in users collection", slog.Any("error", err))
		return nil, ErrInternal
	}
	defer cur.Close(ctx)

	var users []models.User
	if err := cur.All(ctx, &users); err != nil {
		r.logger.Error("failed to extract users from cursor", slog.Any("error", err))
		return nil, ErrInternal
	}

	return users, nil
}
