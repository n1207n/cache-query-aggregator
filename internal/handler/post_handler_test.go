//go:build unit

package handler

import (
	"bytes"
	"encoding/json"
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

func TestPostHandler_CreatePost(t *testing.T) {
	mockService := new(servicemocks.PostService)
	postHandler := NewPostHandler(mockService)

	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		reqBody := CreatePostRequest{
			UserID:  1,
			Content: "This is a great post",
		}
		expectedPost := sqlc.Post{
			ID:        1,
			UserID:    reqBody.UserID,
			Content:   reqBody.Content,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockService.On("CreatePost", mock.Anything, mock.MatchedBy(func(params sqlc.CreatePostParams) bool {
			return params.UserID == reqBody.UserID && params.Content == reqBody.Content
		})).Return(expectedPost, nil).Once()

		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router := gin.Default()
		router.POST("/api/v1/posts", postHandler.CreatePost)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resPost PostResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resPost)
		assert.NoError(t, err)
		assert.Equal(t, expectedPost.ID, resPost.ID)
		assert.Equal(t, expectedPost.Content, resPost.Content)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid_payload", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/posts", bytes.NewBufferString(`{"user_id":1}`)) // Missing content
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router := gin.Default()
		router.POST("/api/v1/posts", postHandler.CreatePost)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestPostHandler_ListPostsByUser(t *testing.T) {
	mockService := new(servicemocks.PostService)
	postHandler := NewPostHandler(mockService)

	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		userID := int64(1)
		expectedPosts := []sqlc.Post{
			{ID: 1, UserID: userID, Content: "Post 1"},
			{ID: 2, UserID: userID, Content: "Post 2"},
		}

		mockService.On("ListPostsByUser", mock.Anything, mock.MatchedBy(func(params sqlc.ListPostsByUserParams) bool {
			return params.UserID == userID
		})).Return(expectedPosts, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/1/posts", nil)
		rr := httptest.NewRecorder()
		router := gin.Default()
		router.GET("/api/v1/users/:id/posts", postHandler.ListPostsByUser)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resPosts []PostResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resPosts)
		assert.NoError(t, err)
		assert.Len(t, resPosts, 2)
		assert.Equal(t, expectedPosts[0].Content, resPosts[0].Content)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid_user_id", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/abc/posts", nil)
		rr := httptest.NewRecorder()
		router := gin.Default()
		router.GET("/api/v1/users/:id/posts", postHandler.ListPostsByUser)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
