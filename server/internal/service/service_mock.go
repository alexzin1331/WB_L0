package service

import "WB_LVL0/server/models"

type MockOrderProvider struct {
	GetOrderFunc func(orderUID string) (*models.Order, error)
}

func (m *MockOrderProvider) GetOrder(orderUID string) (*models.Order, error) {
	return m.GetOrderFunc(orderUID)
}

/*
docker exec -it a39e2ee9b067487f96baf489813f7ac727f89dcef79478cadcdc5b54d6b3bfdd psql -U postgres -d postgres -c "explain (analyze, buffers) SELECT * FROM orders o join deliveries d using (order_uid) where o.delivery_service = 'russianpost' and shard_id = "
docker exec -it a39e2ee9b067487f96baf489813f7ac727f89dcef79478cadcdc5b54d6b3bfdd psql -U postgres -d postgres -c "drop index idx_orders_delivery_service_uid"
CREATE INDEX IF NOT EXISTS idx_deliveries_phone ON deliveries(phone);

docker exec -it a39e2ee9b067487f96baf489813f7ac727f89dcef79478cadcdc5b54d6b3bfdd psql -U postgres -d postgres -c "CREATE INDEX IF NOT EXISTS idx_deliveries_phone ON deliveries(phone);"
"



docker exec -it a39e2ee9b067487f96baf489813f7ac727f89dcef79478cadcdc5b54d6b3bfdd psql -U postgres -d postgres -c ""


docker exec -it a39e2ee9b067487f96baf489813f7ac727f89dcef79478cadcdc5b54d6b3bfdd psql -U postgres -d postgres -c "explain (analyze, buffers) SELECT * FROM orders o join deliveries d using (order_uid) where d.phone = '+3989240939'"

*/
