package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/kaungmyathan18/golang-inventory-app/internal/database"
	"go.uber.org/zap"
)

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	CategoryID  string    `json:"category_id"`
	CreatedAt   time.Time `json:"created_at"`
}
type ProductRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewProductRepository(db *database.DB, log *zap.Logger) *ProductRepository {
	return &ProductRepository{db: db.SQL, log: log}
}

func (r *ProductRepository) Create(ctx context.Context, name, description string, price float64, categoryID string) (*Product, error) {
	category, err := NewCategoryRepository(&database.DB{SQL: r.db}, r.log).Get(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, errors.New("category not found")
	}
	product := Product{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Price:       price,
		CategoryID:  categoryID,
		CreatedAt:   time.Now().UTC(),
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO products (id, name, description, price, category_id, created_at) VALUES (?,?,?,?,?,?)`,
		product.ID, product.Name, product.Description, product.Price, product.CategoryID, product.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) Get(ctx context.Context, id string) (*Product, error) {
	var createdAt string
	product := Product{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, price, category_id, created_at FROM products WHERE id = ?`, id,
	).Scan(&product.ID, &product.Name, &product.Description, &product.Price, &product.CategoryID, &createdAt)
	if err != nil {
		return nil, err
	}
	product.CreatedAt, err = parseCreatedAt(createdAt)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) ListPaged(ctx context.Context, offset, limit int) ([]Product, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description, price, category_id, created_at FROM products ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		var createdAt string
		product := Product{}
		if err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.Price, &product.CategoryID, &createdAt); err != nil {
			return nil, 0, err
		}
		product.CreatedAt, err = parseCreatedAt(createdAt)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, product)
	}
	return products, total, nil
}

func (r *ProductRepository) Update(ctx context.Context, id string, name, description string, price float64, categoryID string) (*Product, error) {
	product := Product{
		ID:          id,
		Name:        name,
		Description: description,
		Price:       price,
		CategoryID:  categoryID,
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE products SET name = ?, description = ?, price = ?, category_id = ? WHERE id = ?`,
		product.Name, product.Description, product.Price, product.CategoryID, product.ID,
	)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return nil
}

type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CategoryRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewCategoryRepository(db *database.DB, log *zap.Logger) *CategoryRepository {
	return &CategoryRepository{db: db.SQL, log: log}
}

func (r *CategoryRepository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM categories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var categories []Category
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.ID, &category.Name); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func (r *CategoryRepository) Get(ctx context.Context, id string) (*Category, error) {
	category := Category{}
	err := r.db.QueryRowContext(ctx, `SELECT id, name FROM categories WHERE id = ?`, id).Scan(&category.ID, &category.Name)
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) Create(ctx context.Context, name string) (*Category, error) {
	category := Category{
		ID:        uuid.NewString(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO categories (id, name, created_at) VALUES (?,?,?)`, category.ID, category.Name, category.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) Update(ctx context.Context, id string, name string) (*Category, error) {
	category := Category{
		ID:   id,
		Name: name,
	}
	_, err := r.db.ExecContext(ctx, `UPDATE categories SET name = ? WHERE id = ?`, category.Name, category.ID)
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return nil
}

type Inventory struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

type InventoryRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewInventoryRepository(db *database.DB, log *zap.Logger) *InventoryRepository {
	return &InventoryRepository{db: db.SQL, log: log}
}

func (r *InventoryRepository) Create(ctx context.Context, productID string, quantity int) (*Inventory, error) {
	inventory := Inventory{
		ID:        uuid.NewString(),
		ProductID: productID,
		Quantity:  quantity,
		CreatedAt: time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO inventory (id, product_id, quantity, created_at) VALUES (?,?,?,?)`, inventory.ID, inventory.ProductID, inventory.Quantity, inventory.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &inventory, nil
}

func (r *InventoryRepository) Get(ctx context.Context, id string) (*Inventory, error) {
	var createdAt string
	inventory := Inventory{}
	err := r.db.QueryRowContext(ctx, `SELECT id, product_id, quantity, created_at FROM inventory WHERE id = ?`, id).Scan(&inventory.ID, &inventory.ProductID, &inventory.Quantity, &createdAt)
	if err != nil {
		return nil, err
	}
	inventory.CreatedAt, err = parseCreatedAt(createdAt)
	if err != nil {
		return nil, err
	}
	return &inventory, nil
}

func (r *InventoryRepository) ListPaged(ctx context.Context, offset, limit int) ([]Inventory, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM inventory`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, product_id, quantity, created_at FROM inventory ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var inventories []Inventory
	for rows.Next() {
		var createdAt string
		inventory := Inventory{}
		if err := rows.Scan(&inventory.ID, &inventory.ProductID, &inventory.Quantity, &createdAt); err != nil {
			return nil, 0, err
		}
		inventory.CreatedAt, err = parseCreatedAt(createdAt)
		if err != nil {
			return nil, 0, err
		}
		inventories = append(inventories, inventory)
	}
	return inventories, total, nil
}

func (r *InventoryRepository) Update(ctx context.Context, id string, productID string, quantity int) (*Inventory, error) {
	inventory := Inventory{
		ID:        id,
		ProductID: productID,
		Quantity:  quantity,
	}
	_, err := r.db.ExecContext(ctx, `UPDATE inventory SET product_id = ?, quantity = ? WHERE id = ?`, inventory.ProductID, inventory.Quantity, inventory.ID)
	if err != nil {
		return nil, err
	}
	return &inventory, nil
}

func (r *InventoryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM inventory WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return nil
}
