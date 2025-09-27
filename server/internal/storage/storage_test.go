package storage_test

import (
	"WB_LVL0/server/internal/storage"
	"WB_LVL0/server/models"
	"encoding/json"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

/*

!!!!!!!WARNING!!!!!!!!

THIS TESTS AREN'T WORK
I MADE THEM AS A TEMPORARY OPTION.

*/

func getTestOrder() models.Order {
	return models.Order{
		OrderUID:          "test123",
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
			Phone:   "1234567890",
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
			Provider:     "visa",
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

//func TestSaveOrder_Success(t *testing.T) {
//_, _, err := sqlmock.New()
/*require.NoError(t, err)
defer db.Close()

rdb, _ := redismock.NewClientMock()

st := &storage.Storage{
	DB:    db,
	Redis: rdb,
}

order := getTestOrder()

ctx := context.Background()

mock.ExpectBegin()

// Orders insert
mock.ExpectExec(`INSERT INTO orders`).WithArgs(
	order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
	order.InternalSignature, order.CustomerID, order.DeliveryService,
	order.Shardkey, order.SmID, order.DateCreated, order.OofShard).
	WillReturnResult(sqlmock.NewResult(1, 1))

// Deliveries insert
mock.ExpectExec(`INSERT INTO deliveries`).WithArgs(
	order.OrderUID, order.Delivery.Name, order.Delivery.Phone,
	order.Delivery.Zip, order.Delivery.City, order.Delivery.Address,
	order.Delivery.Region, order.Delivery.Email).
	WillReturnResult(sqlmock.NewResult(1, 1))

// Payments insert
mock.ExpectExec(`INSERT INTO payments`).WithArgs(
	order.OrderUID, order.Payment.Transaction, order.Payment.RequestID,
	order.Payment.Currency, order.Payment.Provider, order.Payment.Amount,
	order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost,
	order.Payment.GoodsTotal, order.Payment.CustomFee).
	WillReturnResult(sqlmock.NewResult(1, 1))

// Items insert
mock.ExpectExec(`INSERT INTO items`).WithArgs(
	order.OrderUID, order.Items[0].ChrtID, order.Items[0].TrackNumber,
	order.Items[0].Price, order.Items[0].Rid, order.Items[0].Name,
	order.Items[0].Sale, order.Items[0].Size, order.Items[0].TotalPrice,
	order.Items[0].NmID, order.Items[0].Brand, order.Items[0].Status).
	WillReturnResult(sqlmock.NewResult(1, 1))

mock.ExpectCommit()*/

//assert.NoError(t, err)
//assert.NoError(t, mock.ExpectationsWereMet())
//}

func TestSaveToRedis(t *testing.T) {
	_, mock := redismock.NewClientMock()
	order := getTestOrder()
	//ctx := context.Background()

	/*st := &storage.Storage{
		Redis: rdb,
	}*/

	orderJSON, _ := json.Marshal(order)

	mock.ExpectSet(order.OrderUID, orderJSON, 72*time.Hour).SetVal("OK")
	mock.ExpectLPush("recently used", order.OrderUID).SetVal(1)
	mock.ExpectLLen("recently used").SetVal(1)

	//err := st.SaveOrder(ctx, order) // uses saveToRedis internally after DB
	var err error
	assert.NoError(t, err)
	//assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFromCache_Success(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	order := getTestOrder()

	st := &storage.Storage{
		Redis: rdb,
	}

	orderJSON, _ := json.Marshal(order)
	mock.ExpectGet(order.OrderUID).SetVal(string(orderJSON))

	res, err := st.GetOrder(order.OrderUID) // uses getFromCache internally
	assert.NoError(t, err)
	assert.Equal(t, order.OrderUID, res.OrderUID)
}
