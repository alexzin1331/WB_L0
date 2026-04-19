package generator

import (
	"fmt"
	"math/rand"
	"time"

	"WB_LVL0/server/models"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
)

type Address struct {
	City    string
	Address string
	Region  string
}

type Generator struct {
	random *rand.Rand
}

func New(random *rand.Rand) *Generator {
	return &Generator{random: random}
}

func (g *Generator) Order() (models.Order, error) {
	orderUID := uuid.New().String()
	items := g.items()

	fullName := faker.FirstName() + " " + faker.LastName()
	email := faker.Email()

	address := Address{}
	if err := faker.FakeData(&address); err != nil {
		return models.Order{}, fmt.Errorf("generate fake address: %w", err)
	}

	now := time.Now()
	return models.Order{
		OrderUID:    orderUID,
		TrackNumber: fmt.Sprintf("WBIL%08d", g.random.Intn(100000000)),
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    fullName,
			Phone:   "+" + fmt.Sprintf("%d", g.random.Intn(9999999999)),
			Zip:     fmt.Sprintf("%d", g.random.Intn(99999)),
			City:    address.City,
			Address: address.Address,
			Region:  address.Region,
			Email:   email,
		},
		Payment: models.Payment{
			Transaction:  orderUID,
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       g.random.Intn(10000) + 1000,
			PaymentDt:    now.Unix(),
			Bank:         []string{"alpha", "sber", "tinkoff"}[g.random.Intn(3)],
			DeliveryCost: g.random.Intn(2000) + 500,
			GoodsTotal:   g.random.Intn(500) + 100,
			CustomFee:    0,
		},
		Items:             items,
		Locale:            []string{"en", "ru"}[g.random.Intn(2)],
		InternalSignature: "",
		CustomerID:        fmt.Sprintf("user%d", g.random.Intn(1000)),
		DeliveryService:   []string{"meest", "russianpost", "dhl"}[g.random.Intn(3)],
		Shardkey:          fmt.Sprintf("%d", g.random.Intn(10)),
		SmID:              g.random.Intn(100),
		DateCreated:       now,
		OofShard:          fmt.Sprintf("%d", g.random.Intn(5)+1),
	}, nil
}

func (g *Generator) items() []models.Item {
	itemCount := g.random.Intn(3) + 1
	items := make([]models.Item, itemCount)

	for i := 0; i < itemCount; i++ {
		items[i] = models.Item{
			ChrtID:      g.random.Intn(10000000),
			TrackNumber: fmt.Sprintf("TRK%06d", g.random.Intn(1000000)),
			Price:       g.random.Intn(1000) + 100,
			Rid:         uuid.New().String(),
			Name:        faker.Word(),
			Sale:        g.random.Intn(50),
			Size:        fmt.Sprintf("%d", g.random.Intn(10)),
			TotalPrice:  g.random.Intn(500) + 50,
			NmID:        g.random.Intn(10000000),
			Brand:       faker.FirstName() + " " + faker.LastName(),
			Status:      200 + g.random.Intn(3),
		}
	}

	return items
}
