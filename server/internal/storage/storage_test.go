package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"WB_LVL0/server/models"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOrder() models.Order {
	return models.Order{
		OrderUID:          "test-order-123",
		TrackNumber:       "TRACK123",
		Entry:             "entry1",
		Locale:            "en",
		InternalSignature: "sig",
		CustomerID:        "cust123",
		DeliveryService:   "UPS",
		Shardkey:          "shard",
		SmID:              42,
		DateCreated:       time.Now(),
		OofShard:          "oof",
		Delivery: models.Delivery{
			Name:    "John Doe",
			Phone:   "+1234567890",
			Zip:     "12345",
			City:    "City",
			Address: "Some street",
			Region:  "Region",
			Email:   "john@example.com",
		},
		Payment: models.Payment{
			Transaction:  "txn123",
			RequestID:    "req123",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1000,
			PaymentDt:    1638349200,
			Bank:         "Bank",
			DeliveryCost: 100,
			GoodsTotal:   900,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      1,
				TrackNumber: "TRACK123",
				Price:       500,
				Rid:         "rid1",
				Name:        "item1",
				Sale:        0,
				Size:        "M",
				TotalPrice:  500,
				NmID:        1001,
				Brand:       "Brand",
				Status:      202,
			},
		},
	}
}

func TestGetFromCacheSuccess(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	order := testOrder()
	st := &Storage{Redis: rdb}

	orderJSON, err := json.Marshal(order)
	require.NoError(t, err)

	mock.ExpectGet(order.OrderUID).SetVal(string(orderJSON))

	got, err := st.getFromCache(context.Background(), order.OrderUID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	assert.Equal(t, order.OrderUID, got.OrderUID)
}

func TestSaveToRedisSuccess(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	order := testOrder()
	st := &Storage{Redis: rdb}

	orderJSON, err := json.Marshal(order)
	require.NoError(t, err)

	mock.ExpectSet(order.OrderUID, orderJSON, 72*time.Hour).SetVal("OK")
	mock.ExpectLPush(recentOrdersKey, order.OrderUID).SetVal(1)
	mock.ExpectLLen(recentOrdersKey).SetVal(1)

	require.NoError(t, st.saveToRedis(context.Background(), &order))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveToRedisTrimsOverflow(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	order := testOrder()
	st := &Storage{Redis: rdb}
	oldUIDs := []string{"old-order-1", "old-order-2"}

	orderJSON, err := json.Marshal(order)
	require.NoError(t, err)

	mock.ExpectSet(order.OrderUID, orderJSON, 72*time.Hour).SetVal("OK")
	mock.ExpectLPush(recentOrdersKey, order.OrderUID).SetVal(cacheLimit + int64(len(oldUIDs)))
	mock.ExpectLLen(recentOrdersKey).SetVal(cacheLimit + int64(len(oldUIDs)))
	mock.ExpectLRange(recentOrdersKey, cacheLimit, cacheLimit+int64(len(oldUIDs))-1).SetVal(oldUIDs)
	mock.ExpectDel(oldUIDs...).SetVal(int64(len(oldUIDs)))
	mock.ExpectLTrim(recentOrdersKey, 0, cacheLimit-1).SetVal("OK")

	require.NoError(t, st.saveToRedis(context.Background(), &order))
	require.NoError(t, mock.ExpectationsWereMet())
}
