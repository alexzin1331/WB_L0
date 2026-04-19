package service

import (
	"log"
	"net/http"

	"WB_LVL0/server/models"

	"github.com/gin-gonic/gin"
)

type Service struct {
	OrderProvider
}

// OrderProvider is interface that the database implement
type OrderProvider interface {
	GetOrder(orderUID string) (*models.Order, error)
}

func NewService(o OrderProvider) *Service {
	return &Service{o}
}

// GetOrder handler
// @Summary Get order by UID
// @Description Получить заказ по его уникальному идентификатору
// @Tags orders
// @Accept json
// @Produce json
// @Param order_uid path string true "Order UID"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]string
// @Router /order/{order_uid} [get]
func (s *Service) GetOrder(c *gin.Context) {
	orderUID := c.Param("order_uid")
	order, err := s.OrderProvider.GetOrder(orderUID)
	if err != nil {
		log.Printf("get order %q: %v", orderUID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}
