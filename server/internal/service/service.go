package service

import (
	"WB_LVL0/server/models"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
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
func (s *Service) GetOrder(c *gin.Context) {
	orderUID := c.Param("order_uid")
	//get order from PostgreSQL or Redis
	order, err := s.OrderProvider.GetOrder(orderUID)
	if err != nil {
		log.Printf("error of getting order: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error: ": err.Error()})
	}
	c.JSON(http.StatusOK, order)
}
