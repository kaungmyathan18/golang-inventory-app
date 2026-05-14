package service

import (
	"context"

	"github.com/kaungmyathan18/golang-inventory-app/internal/observability"
	"github.com/kaungmyathan18/golang-inventory-app/internal/repository"
	"go.uber.org/zap"
)

type ProductService struct {
	repo    *repository.ProductRepository
	log     *zap.Logger
	metrics *observability.Metrics
}

func NewProductService(
	repo *repository.ProductRepository,
	log *zap.Logger,
	m *observability.Metrics,
) *ProductService {
	return &ProductService{
		repo:    repo,
		log:     log,
		metrics: m,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, name, description string, price float64, categoryID string) (*repository.Product, error) {
	product, err := s.repo.Create(ctx, name, description, price, categoryID)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*repository.Product, error) {
	return s.repo.Get(ctx, id)
}

func (s *ProductService) ListProductsPaged(ctx context.Context, page, limit int) ([]repository.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.repo.ListPaged(ctx, offset, limit)
}

func (s *ProductService) UpdateProduct(ctx context.Context, id string, name, description string, price float64, categoryID string) (*repository.Product, error) {
	product, err := s.repo.Update(ctx, id, name, description, price, categoryID)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ProductService) PingDeps(ctx context.Context) error {
	_ = ctx
	return nil
}

type CategoryService struct {
	repo    *repository.CategoryRepository
	log     *zap.Logger
	metrics *observability.Metrics
}

func NewCategoryService(
	repo *repository.CategoryRepository,
	log *zap.Logger,
	m *observability.Metrics,
) *CategoryService {
	return &CategoryService{
		repo:    repo,
		log:     log,
		metrics: m,
	}
}
func (s *CategoryService) ListCategoriesPaged(ctx context.Context, page, limit int) ([]repository.Category, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.repo.ListPaged(ctx, offset, limit)
}

func (s *CategoryService) GetCategory(ctx context.Context, id string) (*repository.Category, error) {
	return s.repo.Get(ctx, id)
}

func (s *CategoryService) CreateCategory(ctx context.Context, name string) (*repository.Category, error) {
	return s.repo.Create(ctx, name)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id string, name string) (*repository.Category, error) {
	return s.repo.Update(ctx, id, name)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *CategoryService) PingDeps(ctx context.Context) error {
	_ = ctx
	return nil
}

type InventoryService struct {
	repo    *repository.InventoryRepository
	log     *zap.Logger
	metrics *observability.Metrics
}

func NewInventoryService(
	repo *repository.InventoryRepository,
	log *zap.Logger,
	m *observability.Metrics,
) *InventoryService {
	return &InventoryService{
		repo:    repo,
		log:     log,
		metrics: m,
	}
}
func (s *InventoryService) ListInventoriesPaged(ctx context.Context, page, limit int) ([]repository.Inventory, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.repo.ListPaged(ctx, offset, limit)
}

func (s *InventoryService) GetInventory(ctx context.Context, id string) (*repository.Inventory, error) {
	return s.repo.Get(ctx, id)
}

func (s *InventoryService) CreateInventory(ctx context.Context, productID string, quantity int) (*repository.Inventory, error) {
	return s.repo.Create(ctx, productID, quantity)
}

func (s *InventoryService) UpdateInventory(ctx context.Context, id string, productID string, quantity int) (*repository.Inventory, error) {
	return s.repo.Update(ctx, id, productID, quantity)
}

func (s *InventoryService) DeleteInventory(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *InventoryService) PingDeps(ctx context.Context) error {
	_ = ctx
	return nil
}
