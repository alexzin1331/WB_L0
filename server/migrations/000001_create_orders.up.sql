-- Таблица заказов
CREATE TABLE IF NOT EXISTS orders (
    order_uid          VARCHAR(50) PRIMARY KEY,
    track_number       VARCHAR(50) NOT NULL,
    entry              VARCHAR(10) NOT NULL,
    locale             VARCHAR(10) NOT NULL,
    internal_signature VARCHAR(50) DEFAULT '',
    customer_id        VARCHAR(50) NOT NULL,
    delivery_service   VARCHAR(50) NOT NULL,
    shardkey           VARCHAR(10) NOT NULL,
    sm_id              INTEGER NOT NULL,
    date_created       TIMESTAMPTZ NOT NULL,
    oof_shard          VARCHAR(10) NOT NULL
);

-- Таблица доставки
CREATE TABLE IF NOT EXISTS deliveries (
    order_uid VARCHAR(50) PRIMARY KEY REFERENCES orders(order_uid),
    name      VARCHAR(100) NOT NULL,
    phone     VARCHAR(20) NOT NULL,
    zip       VARCHAR(20) NOT NULL,
    city      VARCHAR(100) NOT NULL,
    address   VARCHAR(200) NOT NULL,
    region    VARCHAR(100) NOT NULL,
    email     VARCHAR(100) NOT NULL
);

-- Таблица платежей
CREATE TABLE IF NOT EXISTS payments (
    order_uid     VARCHAR(50) PRIMARY KEY REFERENCES orders(order_uid),
    transaction   VARCHAR(50) NOT NULL,
    request_id    VARCHAR(50) DEFAULT '',
    currency      VARCHAR(10) NOT NULL,
    provider      VARCHAR(50) NOT NULL,
    amount        INTEGER NOT NULL,
    payment_dt    BIGINT NOT NULL,
    bank          VARCHAR(50) NOT NULL,
    delivery_cost INTEGER NOT NULL,
    goods_total   INTEGER NOT NULL,
    custom_fee    INTEGER NOT NULL
);

-- Таблица товаров
CREATE TABLE IF NOT EXISTS items (
    id           SERIAL PRIMARY KEY,
    order_uid    VARCHAR(50) REFERENCES orders(order_uid),
    chrt_id      BIGINT NOT NULL,
    track_number VARCHAR(50) NOT NULL,
    price        INTEGER NOT NULL,
    rid          VARCHAR(50) NOT NULL,
    name         VARCHAR(100) NOT NULL,
    sale         INTEGER NOT NULL,
    size         VARCHAR(10) NOT NULL,
    total_price  INTEGER NOT NULL,
    nm_id        BIGINT NOT NULL,
    brand        VARCHAR(100) NOT NULL,
    status       INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);