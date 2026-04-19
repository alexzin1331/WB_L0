package generator

import (
	"fmt"
	"math/rand"
	"time"

	"WB_LVL0/server/models"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
)

type Generator struct {
	random *rand.Rand
}

var (
	phones           = []string{"+79000000001", "+79000000002", "+79000000003", "+79000000004"}
	customers        = []string{"customer-a", "customer-b", "customer-c", "customer-d"}
	deliveryServices = []string{"meest", "russianpost", "dhl"}
	currencies       = []string{"USD", "EUR", "RUB"}
	banks            = []string{"alpha", "sber", "tinkoff"}
	locales          = []string{"en", "ru"}
	cities           = []string{"Moscow", "Saint Petersburg", "Kazan", "Novosibirsk"}
	regions          = []string{"Moscow", "Leningrad", "Tatarstan", "Siberia"}
	addresses        = []string{"Lenina 1", "Tverskaya 10", "Baumana 7", "Central 15"}
	brands           = []string{"WB Basic", "Nord Line", "City Wear", "Bright Home"}
	itemNames        = []string{"T-Shirt", "Sneakers", "Backpack", "Jacket", "Lamp", "Book"}
)

func New(random *rand.Rand) *Generator {
	return &Generator{random: random}
}

func (g *Generator) Order() (models.Order, error) {
	orderUID := uuid.New().String()
	items := g.items()

	customerID := g.pick(customers)
	cityIndex := g.random.Intn(len(cities))

	now := time.Now()
	return models.Order{
		OrderUID:    orderUID,
		TrackNumber: fmt.Sprintf("WBIL%08d", g.random.Intn(100000000)),
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    faker.FirstName() + " " + faker.LastName(),
			Phone:   g.pick(phones),
			Zip:     fmt.Sprintf("%05d", g.random.Intn(100000)),
			City:    cities[cityIndex],
			Address: addresses[cityIndex],
			Region:  regions[cityIndex],
			Email:   customerID + "@example.com",
		},
		Payment: models.Payment{
			Transaction:  orderUID,
			RequestID:    "",
			Currency:     g.pick(currencies),
			Provider:     "wbpay",
			Amount:       g.random.Intn(10000) + 1000,
			PaymentDt:    now.Unix(),
			Bank:         g.pick(banks),
			DeliveryCost: g.random.Intn(2000) + 500,
			GoodsTotal:   g.random.Intn(500) + 100,
			CustomFee:    0,
		},
		Items:             items,
		Locale:            g.pick(locales),
		InternalSignature: "",
		CustomerID:        customerID,
		DeliveryService:   g.pick(deliveryServices),
		Shardkey:          fmt.Sprintf("%d", g.random.Intn(4)+1),
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
			ChrtID:      g.random.Intn(10000000) + 1,
			TrackNumber: fmt.Sprintf("TRK%06d", g.random.Intn(1000000)),
			Price:       g.random.Intn(1000) + 100,
			Rid:         uuid.New().String(),
			Name:        g.pick(itemNames),
			Sale:        g.random.Intn(50),
			Size:        fmt.Sprintf("%d", g.random.Intn(10)),
			TotalPrice:  g.random.Intn(500) + 50,
			NmID:        g.random.Intn(10000000) + 1,
			Brand:       g.pick(brands),
			Status:      200 + g.random.Intn(3),
		}
	}

	return items
}

func (g *Generator) pick(values []string) string {
	return values[g.random.Intn(len(values))]
}
