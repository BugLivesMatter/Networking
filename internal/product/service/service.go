package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	categoryrepo "github.com/lab2/rest-api/internal/category/repository"
	"github.com/lab2/rest-api/internal/product/domain"
	"github.com/lab2/rest-api/internal/product/dto"
	productrepo "github.com/lab2/rest-api/internal/product/repository"
	"github.com/lab2/rest-api/pkg/apperror"
	"github.com/lab2/rest-api/pkg/pagination"
)

// ProductService определяет бизнес-логику для работы с продуктами.
type ProductService interface {
	// Create создаёт новый продукт.
	// ctx — контекст запроса.
	// req — данные нового продукта (название, цена, категория и т.д.).
	// Возвращает созданный продукт с подгруженной категорией или ошибку.
	Create(ctx context.Context, req *dto.CreateProductRequest) (*domain.Product, error)

	// GetByID возвращает продукт по его UUID.
	// ctx — контекст запроса.
	// id — уникальный идентификатор продукта.
	// Возвращает apperror.ErrNotFound, если продукт не найден или удалён.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)

	// List возвращает постраничный список продуктов.
	// ctx — контекст запроса.
	// page — номер страницы (начиная с 1).
	// limit — количество записей на странице (макс. pagination.MaxLimit).
	// categoryID — опциональный фильтр по UUID категории (nil = без фильтра).
	// Возвращает: срез продуктов, общее число записей, число страниц, ошибку.
	List(ctx context.Context, page, limit int, categoryID *uuid.UUID) ([]domain.Product, int64, int, error)

	// Update полностью заменяет данные продукта.
	// ctx — контекст запроса.
	// id — UUID обновляемого продукта.
	// req — новые данные: все поля обязательны.
	// Возвращает обновлённый продукт с подгруженной категорией или ошибку.
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateProductRequest) (*domain.Product, error)

	// Patch частично обновляет продукт — изменяются только переданные поля.
	// ctx — контекст запроса.
	// id — UUID обновляемого продукта.
	// req — поля для обновления (nil-поля остаются без изменений).
	// Возвращает обновлённый продукт с подгруженной категорией или ошибку.
	Patch(ctx context.Context, id uuid.UUID, req *dto.PatchProductRequest) (*domain.Product, error)

	// Delete выполняет мягкое удаление продукта (soft delete).
	// ctx — контекст запроса.
	// id — UUID удаляемого продукта.
	// Возвращает apperror.ErrNotFound, если продукт не найден.
	Delete(ctx context.Context, id uuid.UUID) error
}

// productService — внутренняя реализация ProductService.
// repo — репозиторий продуктов для операций с БД.
// categoryRepo — репозиторий категорий для валидации существования категории.
type productService struct {
	repo         productrepo.ProductRepository
	categoryRepo categoryrepo.CategoryRepository
}

// NewProductService создаёт новый экземпляр ProductService.
// repo — репозиторий продуктов.
// categoryRepo — репозиторий категорий (используется для проверки categoryID).
func NewProductService(repo productrepo.ProductRepository, categoryRepo categoryrepo.CategoryRepository) ProductService {
	return &productService{repo: repo, categoryRepo: categoryRepo}
}

// Create создаёт новый продукт в БД.
// Проверяет корректность UUID категории и её существование.
// Если статус не указан — устанавливает "available" по умолчанию.
// После создания перечитывает запись, чтобы вернуть данные с подгруженной категорией.
func (s *productService) Create(ctx context.Context, req *dto.CreateProductRequest) (*domain.Product, error) {
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, apperror.ErrBadRequest
	}
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
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
	// Перечитываем, чтобы получить связанную категорию через Preload
	product, _ = s.repo.GetByID(ctx, product.ID)
	return product, nil
}

// GetByID ищет продукт по UUID среди не удалённых записей.
// Возвращает apperror.ErrNotFound, если запись не существует или помечена удалённой.
func (s *productService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	return product, nil
}

// List возвращает постраничный список продуктов.
// Нормализует page и limit: значения ниже минимума заменяются на дефолтные,
// значения выше MaxLimit обрезаются.
// offset вычисляется как (page-1)*limit и передаётся в репозиторий.
func (s *productService) List(ctx context.Context, page, limit int, categoryID *uuid.UUID) ([]domain.Product, int64, int, error) {
	if page < 1 {
		page = pagination.DefaultPage
	}
	if limit < 1 {
		limit = pagination.DefaultLimit
	}
	if limit > pagination.MaxLimit {
		limit = pagination.MaxLimit
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

// Update полностью заменяет все поля продукта.
// Сначала проверяет существование продукта, затем валидирует новую категорию.
// После сохранения перечитывает запись для получения актуальных данных с Preload.
func (s *productService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateProductRequest) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, apperror.ErrBadRequest
	}
	_, err = s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
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

// Patch частично обновляет продукт — применяет только те поля, которые не nil.
// Если передан CategoryID — дополнительно проверяет существование новой категории.
// После сохранения перечитывает запись для получения актуальных данных с Preload.
func (s *productService) Patch(ctx context.Context, id uuid.UUID, req *dto.PatchProductRequest) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	if req.CategoryID != nil {
		categoryID, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			return nil, apperror.ErrBadRequest
		}
		_, err = s.categoryRepo.GetByID(ctx, categoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.ErrNotFound
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

// Delete выполняет мягкое удаление (soft delete) продукта по UUID.
// Сначала проверяет существование записи — если не найдена, возвращает apperror.ErrNotFound.
// Физически запись из БД не удаляется, проставляется поле deleted_at.
func (s *productService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}
