package models

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"regexp"
	"time"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// Config with yaml-tags
type Config struct {
	ServConf ServerCfg   `yaml:"server"`
	DBConf   DatabaseCfg `yaml:"database"`
	RDBConf  Redis       `yaml:"redis"`
}

type Redis struct {
	RedisAddress  string `yaml:"redis_address"`
	RedisPassword string `yaml:"redis_password"`
	RedisDB       int    `yaml:"redis_db"`
}

type ServerCfg struct {
	Timeout time.Duration `yaml:"timeout" env:"TIMEOUT" env-default:"10s"`
	Host    string        `yaml:"hostGateway" env:"HostGateway" env-default:":8081"`
}

type DatabaseCfg struct {
	Port     string `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-default:"1234"`
	DBName   string `yaml:"dbname" env:"DB_NAME" env-default:"postgres"`
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
}

func MustLoad(path string) *Config {
	conf := &Config{}
	if err := cleanenv.ReadConfig(path, conf); err != nil {
		log.Fatal("Can't read the common config")
		return nil
	}
	return conf
}

// Order structs for JSON and DB
type Order struct {
	OrderUID          string    `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	Shardkey          string    `json:"shardkey"`
	SmID              int       `json:"sm_id"`
	DateCreated       time.Time `json:"date_created"`
	OofShard          string    `json:"oof_shard"`
}

type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type Payment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDt    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

type GetOrderRequest struct {
	OrderUID string `json:"order_uid"`
}

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex    = regexp.MustCompile(`^\+\d{5,15}$`)
	trackNumRegex = regexp.MustCompile(`^[A-Z0-9]{8,20}$`)
)

func (o *Order) Validate() error {
	if o.OrderUID == "" {
		return &ValidationError{Field: "order_uid", Message: "is required"}
	}

	if len(o.OrderUID) < 10 || len(o.OrderUID) > 50 {
		return &ValidationError{Field: "order_uid", Message: "must be 10-50 characters"}
	}

	if !trackNumRegex.MatchString(o.TrackNumber) {
		return &ValidationError{Field: "track_number", Message: "invalid format"}
	}

	if o.Entry == "" {
		return &ValidationError{Field: "entry", Message: "is required"}
	}

	// Delivery validation
	if err := o.Delivery.Validate(); err != nil {
		return fmt.Errorf("delivery validation failed: %w", err)
	}

	// Payment validation
	if err := o.Payment.Validate(); err != nil {
		return fmt.Errorf("payment validation failed: %w", err)
	}

	// Items validation
	if len(o.Items) == 0 {
		return &ValidationError{Field: "items", Message: "order must contain at least one item"}
	}

	for i, item := range o.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item %d validation failed: %w", i, err)
		}
	}

	if o.Locale != "en" && o.Locale != "ru" {
		return &ValidationError{Field: "locale", Message: "must be either 'en' or 'ru'"}
	}

	if o.CustomerID == "" {
		return &ValidationError{Field: "customer_id", Message: "is required"}
	}

	if o.DeliveryService == "" {
		return &ValidationError{Field: "delivery_service", Message: "is required"}
	}

	if o.Shardkey == "" {
		return &ValidationError{Field: "shardkey", Message: "is required"}
	}

	if o.SmID < 0 {
		return &ValidationError{Field: "sm_id", Message: "must be positive"}
	}

	if o.DateCreated.IsZero() {
		return &ValidationError{Field: "date_created", Message: "is required"}
	}

	if o.DateCreated.After(time.Now().Add(1 * time.Hour)) {
		return &ValidationError{Field: "date_created", Message: "cannot be in the future"}
	}

	if o.OofShard == "" {
		return &ValidationError{Field: "oof_shard", Message: "is required"}
	}

	return nil
}

func (d *Delivery) Validate() error {
	if d.Name == "" {
		return &ValidationError{Field: "name", Message: "is required"}
	}

	if !phoneRegex.MatchString(d.Phone) {
		return &ValidationError{Field: "phone", Message: "invalid format"}
	}

	if len(d.Zip) < 5 || len(d.Zip) > 20 {
		return &ValidationError{Field: "zip", Message: "must be 5-20 characters"}
	}

	if d.City == "" {
		return &ValidationError{Field: "city", Message: "is required"}
	}

	if d.Address == "" {
		return &ValidationError{Field: "address", Message: "is required"}
	}

	if d.Region == "" {
		return &ValidationError{Field: "region", Message: "is required"}
	}

	if !emailRegex.MatchString(d.Email) {
		return &ValidationError{Field: "email", Message: "invalid format"}
	}

	return nil
}

func (p *Payment) Validate() error {
	if p.Transaction == "" {
		return &ValidationError{Field: "transaction", Message: "is required"}
	}

	if p.Currency != "USD" && p.Currency != "EUR" && p.Currency != "RUB" {
		return &ValidationError{Field: "currency", Message: "invalid currency"}
	}

	if p.Provider != "wbpay" && p.Provider != "applepay" && p.Provider != "googlepay" {
		return &ValidationError{Field: "provider", Message: "invalid payment provider"}
	}

	if p.Amount <= 0 {
		return &ValidationError{Field: "amount", Message: "must be positive"}
	}

	if p.PaymentDt <= 0 {
		return &ValidationError{Field: "payment_dt", Message: "must be positive"}
	}

	if p.Bank == "" {
		return &ValidationError{Field: "bank", Message: "is required"}
	}

	if p.DeliveryCost < 0 {
		return &ValidationError{Field: "delivery_cost", Message: "cannot be negative"}
	}

	if p.GoodsTotal <= 0 {
		return &ValidationError{Field: "goods_total", Message: "must be positive"}
	}

	if p.CustomFee < 0 {
		return &ValidationError{Field: "custom_fee", Message: "cannot be negative"}
	}

	return nil
}

func (i *Item) Validate() error {
	if i.ChrtID <= 0 {
		return &ValidationError{Field: "chrt_id", Message: "must be positive"}
	}

	if !trackNumRegex.MatchString(i.TrackNumber) {
		return &ValidationError{Field: "track_number", Message: "invalid format"}
	}

	if i.Price <= 0 {
		return &ValidationError{Field: "price", Message: "must be positive"}
	}

	if i.Rid == "" {
		return &ValidationError{Field: "rid", Message: "is required"}
	}

	if i.Name == "" {
		return &ValidationError{Field: "name", Message: "is required"}
	}

	if i.Sale < 0 || i.Sale > 100 {
		return &ValidationError{Field: "sale", Message: "must be 0-100"}
	}

	if i.Size == "" {
		return &ValidationError{Field: "size", Message: "is required"}
	}

	if i.TotalPrice <= 0 {
		return &ValidationError{Field: "total_price", Message: "must be positive"}
	}

	if i.NmID <= 0 {
		return &ValidationError{Field: "nm_id", Message: "must be positive"}
	}

	if i.Brand == "" {
		return &ValidationError{Field: "brand", Message: "is required"}
	}

	if i.Status < 0 {
		return &ValidationError{Field: "status", Message: "cannot be negative"}
	}

	return nil
}
