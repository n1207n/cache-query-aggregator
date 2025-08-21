//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	"github.com/n1207n/cache-query-aggregator/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBUserRepository_CreateUser(t *testing.T) {
	userRepo := NewDBUserRepository(testQueries)
	ctx := context.Background()

	email := "test.create." + util.RandomString(6) + "@example.com"
	params := sqlc.CreateUserParams{
		FirstName:      "John",
		LastName:       "Doe",
		Email:          email,
		HashedPassword: "password123", // Repository will hash this
	}

	createdUser, err := userRepo.CreateUser(ctx, params)

	require.NoError(t, err)
	require.NotEmpty(t, createdUser)

	assert.Equal(t, params.FirstName, createdUser.FirstName)
	assert.Equal(t, params.LastName, createdUser.LastName)
	assert.Equal(t, params.Email, createdUser.Email)
	assert.NotEqual(t, params.HashedPassword, createdUser.HashedPassword) // Should be hashed
	assert.NotZero(t, createdUser.ID)
	assert.NotZero(t, createdUser.CreatedAt)
	assert.NotZero(t, createdUser.UpdatedAt)

	// Verify password
	match, err := util.CheckPasswordHash("password123", createdUser.HashedPassword)
	require.NoError(t, err)
	assert.True(t, match)
}

func TestDBUserRepository_GetUserByID(t *testing.T) {
	userRepo := NewDBUserRepository(testQueries)
	ctx := context.Background()

	// First create a user to fetch
	email := "test.get." + util.RandomString(6) + "@example.com"
	params := sqlc.CreateUserParams{
		FirstName:      "Jane",
		LastName:       "Doe",
		Email:          email,
		HashedPassword: "securepassword",
	}
	createdUser, err := userRepo.CreateUser(ctx, params)
	require.NoError(t, err)

	// Now fetch the user by ID
	fetchedUser, err := userRepo.GetUserByID(ctx, createdUser.ID)
	require.NoError(t, err)
	require.NotEmpty(t, fetchedUser)

	assert.Equal(t, createdUser.ID, fetchedUser.ID)
	assert.Equal(t, createdUser.FirstName, fetchedUser.FirstName)
	assert.Equal(t, createdUser.Email, fetchedUser.Email)
}

func TestDBUserRepository_GetUserByEmail(t *testing.T) {
	userRepo := NewDBUserRepository(testQueries)
	ctx := context.Background()

	// First create a user to fetch
	email := "test.getbyemail." + util.RandomString(6) + "@example.com"
	params := sqlc.CreateUserParams{
		FirstName:      "Email",
		LastName:       "User",
		Email:          email,
		HashedPassword: "securepassword",
	}
	createdUser, err := userRepo.CreateUser(ctx, params)
	require.NoError(t, err)

	// Now fetch the user by email
	fetchedUser, err := userRepo.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.NotEmpty(t, fetchedUser)

	assert.Equal(t, createdUser.ID, fetchedUser.ID)
	assert.Equal(t, createdUser.FirstName, fetchedUser.FirstName)
	assert.Equal(t, createdUser.Email, fetchedUser.Email)
}
