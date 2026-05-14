CREATE TABLE IF NOT EXISTS inventory (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NULL,
    deleted_at TEXT NULL,
    CONSTRAINT fk_inventory_product_id FOREIGN KEY (product_id) REFERENCES products (id)
);