//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
)

func TestUserServiceImpl_CreateUser(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	userService := NewUserService(mockRepo)

	ctx := context.Background()
	params := sqlc.CreateUserParams{
		FirstName:      "John",
		LastName:       "Doe",
		Email:          "john.doe@example.com",
		HashedPassword: "password123",
	}
	expectedUser := sqlc.User{
		ID:             1,
		FirstName:      params.FirstName,
		LastName:       params.LastName,
		Email:          params.Email,
		HashedPassword: "hashedpassword",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	mockRepo.On("CreateUser", ctx, params).Return(expectedUser, nil)

	user, err := userService.CreateUser(ctx, params)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}

func TestUserServiceImpl_GetUserByID(t *testing.T) {
	mockRepo := new(mocks.UserRepository)
	userService := NewUserService(mockRepo)

	ctx := context.Background()
	userID := int64(1)
	expectedUser := sqlc.User{
		ID:        userID,
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     "jane.doe@example.com",
	}

	mockRepo.On("GetUserByID", ctx, userID).Return(expectedUser, nil)

	user, err := userService.GetUserByID(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}
