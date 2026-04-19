package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"WB_LVL0/server/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrder_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotUID string
	mockProvider := &MockOrderProvider{
		GetOrderFunc: func(uid string) (*models.Order, error) {
			gotUID = uid
			return &models.Order{OrderUID: "test123"}, nil
		},
	}

	service := NewService(mockProvider)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "order_uid", Value: "test123"}}

	service.GetOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test123", gotUID)
	assert.Contains(t, w.Body.String(), "test123")
}

func TestGetOrder_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockProvider := &MockOrderProvider{
		GetOrderFunc: func(uid string) (*models.Order, error) {
			return nil, errors.New("order not found")
		},
	}

	service := NewService(mockProvider)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "order_uid", Value: "invalid"}}

	service.GetOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "order not found", response["error"])
}
