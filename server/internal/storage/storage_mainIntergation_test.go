package storage

import (
	"WB_LVL0/server/models"
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func setupTestStorage(t *testing.T) *Storage {
	cfg := models.Config{
		DBConf: models.DatabaseCfg{
			Host:     "localhost",
			Port:     "5433",
			User:     "test_user",
			Password: "test_password",
			DBName:   "test_db",
		},
		RDBConf: models.Redis{
			RedisAddress:  "localhost:6379",
			RedisPassword: "",
			RedisDB:       1,
		},
	}

	storage, err := New(cfg)
	require.NoError(t, err)

	// Clear earlier tests
	_, err = storage.db.Exec("DELETE FROM items")
	require.NoError(t, err)
	_, err = storage.db.Exec("DELETE FROM payments")
	require.NoError(t, err)
	_, err = storage.db.Exec("DELETE FROM deliveries")
	require.NoError(t, err)
	_, err = storage.db.Exec("DELETE FROM orders")
	require.NoError(t, err)
	storage.redis.FlushDB(context.Background())

	return storage
}

func cleanupTestStorage(t *testing.T, s *Storage) {
	s.db.Close()
	s.redis.Close()
}

func TestStorage_SaveAndGetOrder(t *testing.T) {
	s := setupTestStorage(t)
	defer cleanupTestStorage(t, s)

	testOrder := models.Order{
		OrderUID:    "test123",
		TrackNumber: "WBIL12345678",
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    "Test User",
			Phone:   "+1234567890",
			Zip:     "12345",
			City:    "Moscow",
			Address: "Test Address",
			Region:  "Test Region",
			Email:   "test@example.com",
		},
		Payment: models.Payment{
			Transaction:  "test123",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1000,
			PaymentDt:    time.Now().Unix(),
			Bank:         "sber",
			DeliveryCost: 500,
			GoodsTotal:   500,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      1234567,
				TrackNumber: "WBIL12345678",
				Price:       100,
				Rid:         "rid123",
				Name:        "Test Item",
				Sale:        10,
				Size:        "1",
				TotalPrice:  90,
				NmID:        1234567,
				Brand:       "Test Brand",
				Status:      200,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test_customer",
		DeliveryService:   "meest",
		Shardkey:          "1",
		SmID:              1,
		DateCreated:       time.Now(),
		OofShard:          "1",
	}

	ctx := context.Background()

	t.Run("SaveOrder success", func(t *testing.T) {
		err := s.SaveOrder(ctx, testOrder)
		require.NoError(t, err)
	})

	t.Run("GetOrder from DB", func(t *testing.T) {
		order, err := s.GetOrder(testOrder.OrderUID)
		require.NoError(t, err)
		require.Equal(t, testOrder.OrderUID, order.OrderUID)
		require.Equal(t, testOrder.Delivery.Name, order.Delivery.Name)
		require.Len(t, order.Items, 1)
	})

	t.Run("GetOrder from Cache", func(t *testing.T) {
		// 1st call GET must download to cache
		_, err := s.GetOrder(testOrder.OrderUID)
		require.NoError(t, err)

		// 2nd call GET must get data from cache
		order, err := s.GetOrder(testOrder.OrderUID)
		require.NoError(t, err)
		require.Equal(t, testOrder.OrderUID, order.OrderUID)
	})

	t.Run("GetOrder not found", func(t *testing.T) {
		_, err := s.GetOrder("nonexistent")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestStorage_PreloadCache(t *testing.T) {
	s := setupTestStorage(t)
	defer cleanupTestStorage(t, s)

	// Create some test orders
	orders := []models.Order{
		{OrderUID: "order1", DateCreated: time.Now()},
		{OrderUID: "order2", DateCreated: time.Now().Add(-time.Hour)},
		{OrderUID: "order3", DateCreated: time.Now().Add(-2 * time.Hour)},
	}

	ctx := context.Background()
	for _, order := range orders {
		err := s.SaveOrder(ctx, order)
		require.NoError(t, err)
	}

	// Check preload
	err := s.preloadCache()
	require.NoError(t, err)

	// Check that data was saved in cache
	for _, uid := range []string{"order1", "order2", "order3"} {
		_, err := s.getFromCache(ctx, uid)
		require.NoError(t, err)
	}
}
