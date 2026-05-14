CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    price REAL NOT NULL,
    created_at TEXT NOT NULL,
    category_id TEXT NOT NULL,
    CONSTRAINT fk_products_category_id FOREIGN KEY (category_id) REFERENCES categories (id)
);