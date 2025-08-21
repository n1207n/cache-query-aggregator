package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/yourusername/yourprojectname/db/sqlc"
	"github.com/yourusername/yourprojectname/internal/util"
)

// UserRepository defines methods for users table
type UserRepository interface {
	CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error)
	GetUserByID(ctx context.Context, id int64) (sqlc.User, error)
	GetUserByEmail(ctx context.Context, email string) (sqlc.User, error)
	ListUsers(ctx context.Context, arg sqlc.ListUsersParams) ([]sqlc.User, error)
	UpdateUser(ctx context.Context, arg sqlc.UpdateUserParams) (sqlc.User, error)
	DeleteUser(ctx context.Context, id int64) error
}

// DBUserRepository takes sqlc.Querier to create an instance
type DBUserRepository struct {
	q sqlc.Querier
}

// NewDBUserRepository creates a new instance of DBUserRepository
func NewDBUserRepository(querier sqlc.Querier) UserRepository {
	return &DBUserRepository{q: querier}
}

// CreateUser creates a new user in DB
// Password gets hashed before save
func (r *DBUserRepository) CreateUser(ctx context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	hashedPassword, err := util.HashPassword(arg.HashedPassword)
	if err != nil {
		return sqlc.User{}, err
	}
	arg.HashedPassword = hashedPassword

	return r.q.CreateUser(ctx, arg)
}

// GetUserByID retrieves a User by id
func (r *DBUserRepository) GetUserByID(ctx context.Context, id int64) (sqlc.User, error) {
	return r.q.GetUserByID(ctx, id)
}

// GetUserByEmail retrieves a User by email
func (r *DBUserRepository) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	return r.q.GetUserByEmail(ctx, email)
}

// ListUsers retrieves a list of Users
func (r *DBUserRepository) ListUsers(ctx context.Context, arg sqlc.ListUsersParams) ([]sqlc.User, error) {
	return r.q.ListUsers(ctx, arg)
}

// UpdateUser updates a User
func (r *DBUserRepository) UpdateUser(ctx context.Context, arg sqlc.UpdateUserParams) (sqlc.User, error) {
	// Rehash password if password gets updated
	if arg.HashedPassword.Valid && arg.HashedPassword.String != "" {
		newHashedPassword, err := util.HashPassword(arg.HashedPassword.String)
		if err != nil {
			return sqlc.User{}, err
		}
		arg.HashedPassword = pgtype.Text{String: newHashedPassword, Valid: true}
	} else {
		if !arg.HashedPassword.Valid {
			// no-op, due to sqlc.narg
		} else if arg.HashedPassword.String == "" {
			arg.HashedPassword.Valid = false
		}
	}

	return r.q.UpdateUser(ctx, arg)
}

// DeleteUser deletes a User by id
func (r *DBUserRepository) DeleteUser(ctx context.Context, id int64) error {
	return r.q.DeleteUser(ctx, id)
}
