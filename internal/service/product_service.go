package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/lab2/rest-api/internal/domain"
	"github.com/lab2/rest-api/internal/dto"
	"github.com/lab2/rest-api/internal/repository"
	"gorm.io/gorm"
)

type ProductService interface {
	Create(ctx context.Context, req *dto.CreateProductRequest) (*domain.Product, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	List(ctx context.Context, page, limit int, categoryID *uuid.UUID) ([]domain.Product, int64, int, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateProductRequest) (*domain.Product, error)
	Patch(ctx context.Context, id uuid.UUID, req *dto.PatchProductRequest) (*domain.Product, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type productService struct {
	repo         repository.ProductRepository
	categoryRepo repository.CategoryRepository
}

func NewProductService(repo repository.ProductRepository, categoryRepo repository.CategoryRepository) ProductService {
	return &productService{repo: repo, categoryRepo: categoryRepo}
}

func (s *productService) Create(ctx context.Context, req *dto.CreateProductRequest) (*domain.Product, error) {
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, ErrBadRequest
	}
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	status := req.Status
	if status == "" {
		status = "available"
	}
	product := &domain.Product{
		CategoryID:  categoryID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Status:      status,
	}
	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}
	product, _ = s.repo.GetByID(ctx, product.ID)
	return product, nil
}

func (s *productService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return product, nil
}

func (s *productService) List(ctx context.Context, page, limit int, categoryID *uuid.UUID) ([]domain.Product, int64, int, error) {
	if page < 1 {
		page = dto.DefaultPage
	}
	if limit < 1 {
		limit = dto.DefaultLimit
	}
	if limit > dto.MaxLimit {
		limit = dto.MaxLimit
	}
	offset := (page - 1) * limit
	products, total, err := s.repo.List(ctx, offset, limit, categoryID)
	if err != nil {
		return nil, 0, 0, err
	}
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return products, total, totalPages, nil
}

func (s *productService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateProductRequest) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, ErrBadRequest
	}
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	product.CategoryID = categoryID
	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.Status = req.Status
	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}
	product, _ = s.repo.GetByID(ctx, id)
	return product, nil
}

func (s *productService) Patch(ctx context.Context, id uuid.UUID, req *dto.PatchProductRequest) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if req.CategoryID != nil {
		categoryID, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			return nil, ErrBadRequest
		}
		_, err = s.categoryRepo.GetByID(ctx, categoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		product.CategoryID = categoryID
	}
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Status != nil {
		product.Status = *req.Status
	}
	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}
	product, _ = s.repo.GetByID(ctx, id)
	return product, nil
}

func (s *productService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}
