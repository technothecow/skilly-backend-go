package usecases

import (
	"context"

	"skilly/internal/domain/models"
	"skilly/internal/domain/repository"
)

func RegisterUser(ctx context.Context, userRepo repository.UserRepository, user models.User) error {
	return userRepo.CreateUser(ctx, user)
}
