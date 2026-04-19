package service

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"WB_LVL0/server/internal/auth"
	"WB_LVL0/server/models"

	"github.com/gin-gonic/gin"
)

type Service struct {
	OrderProvider
	OrderQueries OrderQueryProvider
	UserProvider UserProvider
	AuthManager  *auth.Manager
}

// OrderProvider is interface that the database implement
type OrderProvider interface {
	GetOrder(orderUID string) (*models.Order, error)
}

type OrderQueryProvider interface {
	ListOrderSummaries(ctx context.Context, filters models.OrderFilters) ([]models.OrderSummary, error)
	AggregateOrders(ctx context.Context, groupBy string, filters models.OrderFilters) ([]models.OrderAggregation, error)
	GetOrderFilterValues(ctx context.Context) (*models.OrderFilterValues, error)
}

type UserProvider interface {
	CreateUser(ctx context.Context, username, passwordHash string) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByID(ctx context.Context, userID int64) (*models.User, error)
}

func NewService(provider OrderProvider, authManagers ...*auth.Manager) *Service {
	service := &Service{OrderProvider: provider}
	if queries, ok := provider.(OrderQueryProvider); ok {
		service.OrderQueries = queries
	}
	if users, ok := provider.(UserProvider); ok {
		service.UserProvider = users
	}
	if len(authManagers) > 0 {
		service.AuthManager = authManagers[0]
	}
	return service
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

func (s *Service) Register(c *gin.Context) {
	if s.UserProvider == nil || s.AuthManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth is not configured"})
		return
	}

	request, ok := parseAuthRequest(c)
	if !ok {
		return
	}

	passwordHash, err := auth.HashPassword(request.Password)
	if err != nil {
		log.Printf("hash password for %q: %v", request.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	user, err := s.UserProvider.CreateUser(c.Request.Context(), request.Username, passwordHash)
	if err != nil {
		log.Printf("create user %q: %v", request.Username, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "user already exists or invalid"})
		return
	}

	token, err := s.AuthManager.Generate(user.ID, user.Username)
	if err != nil {
		log.Printf("generate token for %q: %v", user.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	c.JSON(http.StatusCreated, models.AuthResponse{Token: token, User: *user})
}

func (s *Service) Login(c *gin.Context) {
	if s.UserProvider == nil || s.AuthManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "auth is not configured"})
		return
	}

	request, ok := parseAuthRequest(c)
	if !ok {
		return
	}

	user, err := s.UserProvider.GetUserByUsername(c.Request.Context(), request.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	if err := auth.CheckPassword(user.PasswordHash, request.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	token, err := s.AuthManager.Generate(user.ID, user.Username)
	if err != nil {
		log.Printf("generate token for %q: %v", user.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: *user})
}

func (s *Service) Me(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok || s.UserProvider == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := s.UserProvider.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (s *Service) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.AuthManager == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth is not configured"})
			return
		}

		token, err := auth.BearerToken(c.GetHeader("Authorization"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		claims, err := s.AuthManager.Parse(token)
		if err != nil {
			status := http.StatusUnauthorized
			message := "invalid token"
			if errors.Is(err, auth.ErrExpiredToken) {
				message = "token expired"
			}
			c.AbortWithStatusJSON(status, gin.H{"error": message})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func (s *Service) ListOrders(c *gin.Context) {
	if s.OrderQueries == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "order queries are not configured"})
		return
	}

	orders, err := s.OrderQueries.ListOrderSummaries(c.Request.Context(), parseOrderFilters(c))
	if err != nil {
		log.Printf("list orders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (s *Service) AggregateOrders(c *gin.Context) {
	if s.OrderQueries == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "order queries are not configured"})
		return
	}

	groupBy := c.DefaultQuery("group_by", "delivery_service")
	aggregations, err := s.OrderQueries.AggregateOrders(c.Request.Context(), groupBy, parseOrderFilters(c))
	if err != nil {
		log.Printf("aggregate orders by %q: %v", groupBy, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"group_by": groupBy, "items": aggregations})
}

func (s *Service) GetFilterValues(c *gin.Context) {
	if s.OrderQueries == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "order queries are not configured"})
		return
	}

	values, err := s.OrderQueries.GetOrderFilterValues(c.Request.Context())
	if err != nil {
		log.Printf("get filter values: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load filter values"})
		return
	}

	c.JSON(http.StatusOK, values)
}

func parseAuthRequest(c *gin.Context) (models.AuthRequest, bool) {
	var request models.AuthRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json body"})
		return request, false
	}

	request.Username = strings.TrimSpace(request.Username)
	if len(request.Username) < 3 || len(request.Username) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username must be 3-64 characters"})
		return request, false
	}
	if len(request.Password) < 6 || len(request.Password) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be 6-128 characters"})
		return request, false
	}

	return request, true
}

func parseOrderFilters(c *gin.Context) models.OrderFilters {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	return models.OrderFilters{
		Phone:           strings.TrimSpace(c.Query("phone")),
		CustomerID:      strings.TrimSpace(c.Query("customer_id")),
		DeliveryService: strings.TrimSpace(c.Query("delivery_service")),
		TrackNumber:     strings.TrimSpace(c.Query("track_number")),
		Bank:            strings.TrimSpace(c.Query("bank")),
		Currency:        strings.TrimSpace(c.Query("currency")),
		Locale:          strings.TrimSpace(c.Query("locale")),
		City:            strings.TrimSpace(c.Query("city")),
		Region:          strings.TrimSpace(c.Query("region")),
		Shardkey:        strings.TrimSpace(c.Query("shardkey")),
		Limit:           limit,
	}
}

func currentUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := value.(int64)
	return userID, ok && userID > 0
}
