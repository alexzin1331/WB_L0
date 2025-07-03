package storage

import (
	"WB_LVL0/server/models"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"log"
	"sync"
	"time"
)

const (
	migrationPath = "file://migrations"
	//migrationPath = "file://server/migrations" -- local
)

const (
	cacheLimit = 1000
)

type Storage struct {
	db    *sql.DB
	redis *redis.Client
}

func initRedis(config models.Config) (*redis.Client, error) {
	var rdb *redis.Client
	rdb = redis.NewClient(&redis.Options{
		Addr:     config.RDBConf.RedisAddress,
		Password: config.RDBConf.RedisPassword,
		DB:       config.RDBConf.RedisDB,
	})
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}
	return rdb, nil
}

func runMigrations(db *sql.DB) error {
	const op = "storage.migrations"
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err = m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			return fmt.Errorf("%s: %w", op, err)
		}
		log.Println("No migrations to apply.")
	} else {
		log.Println("Database migrations applied successfully.")
	}
	return nil
}

// New create new storage with Redis and Postgres
func New(c models.Config) (*Storage, error) {
	const op = "storage.connection"
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", c.DBConf.Host, c.DBConf.Port, c.DBConf.User, c.DBConf.Password, c.DBConf.DBName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err = waitForDB(db, 5, 1*time.Second); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	log.Println("Connection is ready")
	rdb, err := initRedis(c)
	if err != nil {
		return nil, fmt.Errorf("%s (initRedis): %w", op, err)
	}

	s := &Storage{
		db:    db,
		redis: rdb,
	}
	if err = runMigrations(db); err != nil {
		return &Storage{}, fmt.Errorf("failed to make migrations: %w", err)
	}
	log.Printf("\n\nmigraitions is success\n\n")
	if err := s.preloadCache(); err != nil {
		log.Printf("%s: %w", op, err)
	}
	return s, nil
}

// waitForDB attempts to reconnect to the database.
// This is necessary because when running in Docker,
// the server might try to connect before the database is fully initialized.
func waitForDB(db *sql.DB, attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		err := db.Ping()
		if err == nil {
			return nil
		}
		log.Printf("Waiting for DB... attempt %d/%d: %v", i+1, attempts, err)
		time.Sleep(delay)
	}
	return fmt.Errorf("database is not reachable after %d attempts", attempts)
}

// preloadCache loads the most recent order UIDs from the database (up to cacheLimit)
// and initiates their preloading into Redis cache.
// Note: Individual scan/load errors are logged but don't stop the process.
func (s *Storage) preloadCache() error {
	const op = "storage.preloadCache"
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `SELECT order_uid FROM orders ORDER BY date_created DESC LIMIT $1`, cacheLimit)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	orderUids := make([]string, 0)
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			log.Printf("%s: %w", op, err)
			continue
		}
		orderUids = append(orderUids, uid)
	}
	if len(orderUids) > 0 {
		s.batchPreload(orderUids)
	}
	return nil
}

// batchPreload efficiently preloads multiple orders into Redis using concurrent workers.
// Features:
// -Limits concurrency using a semaphore (max 'size' goroutines)
// -Uses wait group to ensure all preloads complete
// -Each order:
//  1. Fetches from database
//  2. Saves to Redis with 2-second timeout
//
// Errors are logged per-order but don't stop the batch.
func (s *Storage) batchPreload(uids []string) {
	const size = 50
	sem := make(chan struct{}, size)
	wg := &sync.WaitGroup{}
	for _, uid := range uids {
		sem <- struct{}{}
		wg.Add(1)
		go func(uid string) {
			defer func() {
				<-sem
				wg.Done()
			}()
			order, err := s.getFromDB(uid)
			if err != nil {
				log.Printf("Preload get order error (UID: %s): %v", uid, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := s.saveToRedis(ctx, order); err != nil {
				log.Printf("(Preload) save order to redis error (UID: %s): %v", uid, err)
			}
		}(uid)
	}
	wg.Wait()
}

// SaveOrder save order in PostgreSQL
func (s *Storage) SaveOrder(ctx context.Context, order models.Order) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			log.Printf("transaction rolled back: %v", err)
		}
	}()

	// 1. Save main order
	orderQuery := `INSERT INTO orders (
		order_uid, track_number, entry, locale, internal_signature, 
		customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = tx.ExecContext(ctx, orderQuery,
		order.OrderUID,
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.CustomerID,
		order.DeliveryService,
		order.Shardkey,
		order.SmID,
		order.DateCreated,
		order.OofShard,
	)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// 2. Save main
	deliveryQuery := `INSERT INTO deliveries (
		order_uid, name, phone, zip, city, address, region, email
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = tx.ExecContext(ctx, deliveryQuery,
		order.OrderUID,
		order.Delivery.Name,
		order.Delivery.Phone,
		order.Delivery.Zip,
		order.Delivery.City,
		order.Delivery.Address,
		order.Delivery.Region,
		order.Delivery.Email,
	)
	if err != nil {
		return fmt.Errorf("failed to insert delivery: %w", err)
	}

	// 3. Save payment
	paymentQuery := `INSERT INTO payments (
		order_uid, transaction, request_id, currency, provider, 
		amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err = tx.ExecContext(ctx, paymentQuery,
		order.OrderUID,
		order.Payment.Transaction,
		order.Payment.RequestID,
		order.Payment.Currency,
		order.Payment.Provider,
		order.Payment.Amount,
		order.Payment.PaymentDt,
		order.Payment.Bank,
		order.Payment.DeliveryCost,
		order.Payment.GoodsTotal,
		order.Payment.CustomFee,
	)
	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}

	// 4. Save items
	itemQuery := `INSERT INTO items (
		order_uid, chrt_id, track_number, price, rid, name, 
		sale, size, total_price, nm_id, brand, status
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, itemQuery,
			order.OrderUID,
			item.ChrtID,
			item.TrackNumber,
			item.Price,
			item.Rid,
			item.Name,
			item.Sale,
			item.Size,
			item.TotalPrice,
			item.NmID,
			item.Brand,
			item.Status,
		)
		if err != nil {
			return fmt.Errorf("failed to insert item: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Order %s saved successfully", order.OrderUID)
	return nil
}

// get data from redis
func (s *Storage) getFromCache(ctx context.Context, orderUID string) (*models.Order, error) {
	val, err := s.redis.Get(ctx, orderUID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("not found in cache")
		}
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var order models.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, fmt.Errorf("cache decode error: %w", err)
	}

	return &order, nil
}

// GetOrder retrieves an order by its UID using cache-first strategy:
// 1. First attempts to fetch from Redis cache
// 2. On cache miss, falls back to database
// 3. On successful DB fetch, repopulates cache
func (s *Storage) GetOrder(orderUID string) (*models.Order, error) {
	cachedOrder, err := s.getFromCache(context.Background(), orderUID)
	//the special message that the data is taken from the cache!
	if err == nil {
		log.Printf("-------------\nget from cache success\n---------------")
		return cachedOrder, nil
	}
	order, err := s.getFromDB(orderUID)
	if err != nil {
		return nil, fmt.Errorf("error of getting order from DB: %v", err)
	}
	if err = s.saveToRedis(context.Background(), order); err != nil {
		return nil, fmt.Errorf("failed to save data in redis: %v", err)
	}
	return order, nil
}

// get data from PostgreSQL
func (s *Storage) getFromDB(orderUID string) (*models.Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	order := models.Order{OrderUID: orderUID}
	orderQuery := `SELECT 
		track_number, entry, locale, internal_signature, customer_id, 
		delivery_service, shardkey, sm_id, date_created, oof_shard 
	FROM orders WHERE order_uid = $1`

	err = tx.QueryRow(orderQuery, orderUID).Scan(
		&order.TrackNumber,
		&order.Entry,
		&order.Locale,
		&order.InternalSignature,
		&order.CustomerID,
		&order.DeliveryService,
		&order.Shardkey,
		&order.SmID,
		&order.DateCreated,
		&order.OofShard,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// 2. receiving delivery data
	delivery := models.Delivery{}
	deliveryQuery := `SELECT 
		name, phone, zip, city, address, region, email 
	FROM deliveries WHERE order_uid = $1`

	err = tx.QueryRow(deliveryQuery, orderUID).Scan(
		&delivery.Name,
		&delivery.Phone,
		&delivery.Zip,
		&delivery.City,
		&delivery.Address,
		&delivery.Region,
		&delivery.Email,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}
	order.Delivery = delivery

	// 3. receiving payment data
	payment := models.Payment{}
	paymentQuery := `SELECT 
		transaction, request_id, currency, provider, amount, 
		payment_dt, bank, delivery_cost, goods_total, custom_fee 
	FROM payments WHERE order_uid = $1`

	err = tx.QueryRow(paymentQuery, orderUID).Scan(
		&payment.Transaction,
		&payment.RequestID,
		&payment.Currency,
		&payment.Provider,
		&payment.Amount,
		&payment.PaymentDt,
		&payment.Bank,
		&payment.DeliveryCost,
		&payment.GoodsTotal,
		&payment.CustomFee,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	order.Payment = payment

	// 4. receiving items data
	itemsQuery := `SELECT 
		chrt_id, track_number, price, rid, name, sale, 
		size, total_price, nm_id, brand, status 
	FROM items WHERE order_uid = $1`

	rows, err := tx.Query(itemsQuery, orderUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		err = rows.Scan(
			&item.ChrtID,
			&item.TrackNumber,
			&item.Price,
			&item.Rid,
			&item.Name,
			&item.Sale,
			&item.Size,
			&item.TotalPrice,
			&item.NmID,
			&item.Brand,
			&item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items: %w", err)
	}
	order.Items = items

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &order, nil
}

// saveToRedis stores an order in Redis with two-phase caching:
// 1. Primary storage: Order JSON stored as key-value with 72-hour TTL
// 2. LRU tracking: Order UID added to "recently used" list for cache management
//
// Performs automatic cache maintenance:
// - Trims "recently used" list when exceeding cacheLimit
// - Removes associated order data when trimming
func (s *Storage) saveToRedis(ctx context.Context, order *models.Order) error {
	orderJSON, err := json.Marshal(order)
	const Lkey = "recently used"
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	if err = s.redis.Set(ctx, order.OrderUID, orderJSON, 72*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}
	if err = s.redis.LPush(ctx, Lkey, order.OrderUID).Err(); err != nil {
		return fmt.Errorf("redis lpush error: %w", err)
	}
	length, err := s.redis.LLen(ctx, Lkey).Result()
	if err != nil {
		return fmt.Errorf("redis llen error: %w", err)
	}
	if length > cacheLimit {
		olds, err := s.redis.LRange(ctx, Lkey, int64(cacheLimit), length-1).Result()
		if err != nil {
			return fmt.Errorf("redis lrange error: %w", err)
		}
		if err := s.redis.Del(ctx, olds...).Err(); err != nil {
			return fmt.Errorf("redis del error: %w", err)
		}
		if err := s.redis.LTrim(ctx, Lkey, 0, int64(cacheLimit)-1).Err(); err != nil {
			return fmt.Errorf("redis ltrim error: %w", err)
		}
	}
	return nil
}
