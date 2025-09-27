package service

import (
	"WB_LVL0/server/models"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOrder_Success(t *testing.T) {
	mockProvider := &MockOrderProvider{
		GetOrderFunc: func(uid string) (*models.Order, error) {
			return &models.Order{OrderUID: "test123"}, nil
		},
	}

	service := NewService(mockProvider)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "order_uid", Value: "test123"}}

	service.GetOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test123")
}

func TestGetOrder_Error(t *testing.T) {
	mockProvider := &MockOrderProvider{
		GetOrderFunc: func(uid string) (*models.Order, error) {
			return nil, errors.New("order not found")
		},
	}

	service := NewService(mockProvider)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "order_uid", Value: "invalid"}}

	service.GetOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}
