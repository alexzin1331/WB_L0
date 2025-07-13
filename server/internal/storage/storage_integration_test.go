package storage

import (
	"WB_LVL0/server/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/require"
)

func TestGetFromCache(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	storage := &Storage{redis: rdb}

	testOrder := models.Order{OrderUID: "test123"}

	t.Run("success", func(t *testing.T) {
		orderJSON := `{"order_uid":"test123"}`
		mock.ExpectGet("test123").SetVal(orderJSON)

		order, err := storage.getFromCache(context.Background(), "test123")
		require.NoError(t, err)
		require.Equal(t, testOrder.OrderUID, order.OrderUID)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectGet("notfound").RedisNil()

		_, err := storage.getFromCache(context.Background(), "notfound")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found in cache")
	})

	t.Run("invalid data", func(t *testing.T) {
		mock.ExpectGet("invalid").SetVal("invalid json")

		_, err := storage.getFromCache(context.Background(), "invalid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cache decode error")
	})
}

func TestSaveToRedis(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	storage := &Storage{redis: rdb}

	testOrder := models.Order{
		OrderUID: "test123",
		Delivery: models.Delivery{
			Name: "Test User",
		},
	}

	// Функция для получения ожидаемого JSON
	getExpectedJSON := func(order *models.Order) []byte {
		jsonData, err := json.Marshal(order)
		if err != nil {
			t.Fatalf("Failed to marshal test order: %v", err)
		}
		return (jsonData)
	}

	t.Run("success", func(t *testing.T) {
		expectedJSON := getExpectedJSON(&testOrder)

		mock.ExpectSet("test123", expectedJSON, 72*time.Hour).SetVal("OK")
		mock.ExpectLPush("recently used", "test123").SetVal(1)
		mock.ExpectLLen("recently used").SetVal(1)

		err := storage.saveToRedis(context.Background(), &testOrder)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cache limit exceeded", func(t *testing.T) {
		expectedJSON := getExpectedJSON(&testOrder)

		mock.ExpectSet("test123", expectedJSON, 72*time.Hour).SetVal("OK")
		mock.ExpectLPush("recently used", "test123").SetVal(1)
		mock.ExpectLLen("recently used").SetVal(1001)
		mock.ExpectLRange("recently used", 1000, 1000).SetVal([]string{"old1"})
		mock.ExpectDel("old1").SetVal(1)
		mock.ExpectLTrim("recently used", 0, 999).SetVal("OK")

		err := storage.saveToRedis(context.Background(), &testOrder)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetFromDB(t *testing.T) {
	fmt.Println("wegergergeg;gme/\n\n\n\n\rhkrp[hr")
	db, mock, err := sqlmock.New()

	require.NoError(t, err)
	fmt.Println("klmer;gme/\n\n\n\n\rhaegresrg")
	defer db.Close()

	storage := &Storage{db: db}

	t.Run("success", func(t *testing.T) {
		// Настройка моков для всех запросов
		orderRows := sqlmock.NewRows([]string{
			"track_number", "entry", "locale", "internal_signature", "customer_id",
			"delivery_service", "shardkey", "sm_id", "date_created", "oof_shard",
		}).AddRow(
			"WBIL12345678", "WBIL", "en", "", "test_customer",
			"meest", "1", 1, time.Now(), "1",
		)

		deliveryRows := sqlmock.NewRows([]string{
			"name", "phone", "zip", "city", "address", "region", "email",
		}).AddRow(
			"Test User", "+1234567890", "12345", "Moscow", "Test Address", "Test Region", "test@example.com",
		)

		paymentRows := sqlmock.NewRows([]string{
			"transaction", "request_id", "currency", "provider", "amount",
			"payment_dt", "bank", "delivery_cost", "goods_total", "custom_fee",
		}).AddRow(
			"test123", "", "USD", "wbpay", 1000,
			time.Now().Unix(), "sber", 500, 500, 0,
		)

		itemRows := sqlmock.NewRows([]string{
			"chrt_id", "track_number", "price", "rid", "name", "sale",
			"size", "total_price", "nm_id", "brand", "status",
		}).AddRow(
			1234567, "WBIL12345678", 100, "rid123", "Test Item", 10,
			"1", 90, 1234567, "Test Brand", 200,
		)

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT.*FROM orders").WillReturnRows(orderRows)
		mock.ExpectQuery("SELECT.*FROM deliveries").WillReturnRows(deliveryRows)
		mock.ExpectQuery("SELECT.*FROM payments").WillReturnRows(paymentRows)
		mock.ExpectQuery("SELECT.*FROM items").WillReturnRows(itemRows)
		mock.ExpectCommit()

		order, err := storage.getFromDB("test123")
		require.NoError(t, err)
		require.Equal(t, "test123", order.OrderUID)
		require.Equal(t, "Test User", order.Delivery.Name)
		require.Len(t, order.Items, 1)
	})

	t.Run("order not found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT.*FROM orders").WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		_, err := storage.getFromDB("notfound")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}
