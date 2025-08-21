//go:build unit

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/n1207n/cache-query-aggregator/db/sqlc"
	servicemocks "github.com/n1207n/cache-query-aggregator/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserHandler_CreateUser(t *testing.T) {
	mockService := new(servicemocks.UserService)
	userHandler := NewUserHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/users", userHandler.CreateUser)

	t.Run("success", func(t *testing.T) {
		reqBody := CreateUserRequest{
			FirstName: "Test",
			LastName:  "User",
			Email:     "test@example.com",
			Password:  "password123",
		}
		expectedUser := sqlc.User{
			ID:        1,
			FirstName: reqBody.FirstName,
			LastName:  reqBody.LastName,
			Email:     reqBody.Email,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockService.On("CreateUser", mock.Anything, mock.MatchedBy(func(params sqlc.CreateUserParams) bool {
			return params.FirstName == reqBody.FirstName && params.Email == reqBody.Email && params.HashedPassword == reqBody.Password
		})).Return(expectedUser, nil).Once()

		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resUser UserResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resUser)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, resUser.ID)
		assert.Equal(t, expectedUser.FirstName, resUser.FirstName)
		assert.Equal(t, expectedUser.Email, resUser.Email)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid_payload", func(t *testing.T) {
		// Missing email
		reqBody := `{"first_name": "Test", "last_name": "User", "password": "password123"}`
		req, _ := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestUserHandler_GetUserByID(t *testing.T) {
	mockService := new(servicemocks.UserService)
	userHandler := NewUserHandler(mockService)

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/users/:id", userHandler.GetUserByID)

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		expectedUser := sqlc.User{
			ID:        userID,
			FirstName: "Test",
			LastName:  "User",
			Email:     "test@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockService.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", userID), nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resUser UserResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resUser)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, resUser.ID)
		assert.Equal(t, expectedUser.FirstName, resUser.FirstName)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid_id", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/users/abc", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
