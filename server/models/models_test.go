package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validOrder() Order {
	return Order{
		OrderUID:          "test-order-123",
		TrackNumber:       "TRACK123",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "customer-1",
		DeliveryService:   "dhl",
		Shardkey:          "1",
		SmID:              10,
		DateCreated:       time.Now(),
		OofShard:          "1",
		Delivery: Delivery{
			Name:    "John Doe",
			Phone:   "+1234567890",
			Zip:     "12345",
			City:    "City",
			Address: "Some street",
			Region:  "Region",
			Email:   "john@example.com",
		},
		Payment: Payment{
			Transaction:  "test-order-123",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1000,
			PaymentDt:    time.Now().Unix(),
			Bank:         "alpha",
			DeliveryCost: 100,
			GoodsTotal:   900,
			CustomFee:    0,
		},
		Items: []Item{
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

func TestOrderValidateSuccess(t *testing.T) {
	order := validOrder()

	require.NoError(t, order.Validate())
}

func TestOrderValidateErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Order)
		wantErr string
	}{
		{
			name: "missing order uid",
			mutate: func(order *Order) {
				order.OrderUID = ""
			},
			wantErr: "order_uid",
		},
		{
			name: "invalid delivery email",
			mutate: func(order *Order) {
				order.Delivery.Email = "bad-email"
			},
			wantErr: "email",
		},
		{
			name: "invalid payment amount",
			mutate: func(order *Order) {
				order.Payment.Amount = 0
			},
			wantErr: "amount",
		},
		{
			name: "empty items",
			mutate: func(order *Order) {
				order.Items = nil
			},
			wantErr: "items",
		},
		{
			name: "future date",
			mutate: func(order *Order) {
				order.DateCreated = time.Now().Add(2 * time.Hour)
			},
			wantErr: "date_created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := validOrder()
			tt.mutate(&order)

			err := order.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
