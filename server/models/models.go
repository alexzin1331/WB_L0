package models

import (
	"errors"
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
		return errors.New("order_uid is required")
	}

	if len(o.OrderUID) < 10 || len(o.OrderUID) > 50 {
		return errors.New("order_uid must be 10-50 characters")
	}

	if !trackNumRegex.MatchString(o.TrackNumber) {
		return errors.New("invalid track_number format")
	}

	if o.Entry == "" {
		return errors.New("entry is required")
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
		return errors.New("order must contain at least one item")
	}

	for i, item := range o.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item %d validation failed: %w", i, err)
		}
	}

	if o.Locale != "en" && o.Locale != "ru" {
		return errors.New("locale must be either 'en' or 'ru'")
	}

	if o.CustomerID == "" {
		return errors.New("customer_id is required")
	}

	if o.DeliveryService == "" {
		return errors.New("delivery_service is required")
	}

	if o.Shardkey == "" {
		return errors.New("shardkey is required")
	}

	if o.SmID < 0 {
		return errors.New("sm_id must be positive")
	}

	if o.DateCreated.IsZero() {
		return errors.New("date_created is required")
	}

	if o.DateCreated.After(time.Now().Add(1 * time.Hour)) {
		return errors.New("date_created cannot be in the future")
	}

	if o.OofShard == "" {
		return errors.New("oof_shard is required")
	}

	return nil
}

func (d *Delivery) Validate() error {
	if d.Name == "" {
		return errors.New("name is required")
	}

	if !phoneRegex.MatchString(d.Phone) {
		return errors.New("invalid phone format")
	}

	if len(d.Zip) < 5 || len(d.Zip) > 20 {
		return errors.New("zip must be 5-20 characters")
	}

	if d.City == "" {
		return errors.New("city is required")
	}

	if d.Address == "" {
		return errors.New("address is required")
	}

	if d.Region == "" {
		return errors.New("region is required")
	}

	if !emailRegex.MatchString(d.Email) {
		return errors.New("invalid email format")
	}

	return nil
}

func (p *Payment) Validate() error {
	if p.Transaction == "" {
		return errors.New("transaction is required")
	}

	if p.Currency != "USD" && p.Currency != "EUR" && p.Currency != "RUB" {
		return errors.New("invalid currency")
	}

	if p.Provider != "wbpay" && p.Provider != "applepay" && p.Provider != "googlepay" {
		return errors.New("invalid payment provider")
	}

	if p.Amount <= 0 {
		return errors.New("amount must be positive")
	}

	if p.PaymentDt <= 0 {
		return errors.New("payment_dt must be positive")
	}

	if p.Bank == "" {
		return errors.New("bank is required")
	}

	if p.DeliveryCost < 0 {
		return errors.New("delivery_cost cannot be negative")
	}

	if p.GoodsTotal <= 0 {
		return errors.New("goods_total must be positive")
	}

	if p.CustomFee < 0 {
		return errors.New("custom_fee cannot be negative")
	}

	return nil
}

func (i *Item) Validate() error {
	if i.ChrtID <= 0 {
		return errors.New("chrt_id must be positive")
	}

	if !trackNumRegex.MatchString(i.TrackNumber) {
		return errors.New("invalid track_number format")
	}

	if i.Price <= 0 {
		return errors.New("price must be positive")
	}

	if i.Rid == "" {
		return errors.New("rid is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	if i.Sale < 0 || i.Sale > 100 {
		return errors.New("sale must be 0-100")
	}

	if i.Size == "" {
		return errors.New("size is required")
	}

	if i.TotalPrice <= 0 {
		return errors.New("total_price must be positive")
	}

	if i.NmID <= 0 {
		return errors.New("nm_id must be positive")
	}

	if i.Brand == "" {
		return errors.New("brand is required")
	}

	if i.Status < 0 {
		return errors.New("status cannot be negative")
	}

	return nil
}
