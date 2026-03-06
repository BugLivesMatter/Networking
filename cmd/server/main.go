package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lab2/rest-api/internal/config"
	"github.com/lab2/rest-api/internal/domain"
	"github.com/lab2/rest-api/internal/handler"
	"github.com/lab2/rest-api/internal/middleware"
	"github.com/lab2/rest-api/internal/repository"
	"github.com/lab2/rest-api/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	if err := db.AutoMigrate(&domain.Category{}, &domain.Product{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	categorySvc := service.NewCategoryService(categoryRepo, productRepo)
	productSvc := service.NewProductService(productRepo, categoryRepo)
	categoryHandler := handler.NewCategoryHandler(categorySvc)
	productHandler := handler.NewProductHandler(productSvc)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.Recovery())

	categories := r.Group("/categories")
	{
		categories.GET("", categoryHandler.List)
		categories.GET("/:id", categoryHandler.GetByID)
		categories.POST("", categoryHandler.Create)
		categories.PUT("/:id", categoryHandler.Update)
		categories.PATCH("/:id", categoryHandler.Patch)
		categories.DELETE("/:id", categoryHandler.Delete)
	}

	products := r.Group("/products")
	{
		products.GET("", productHandler.List)
		products.GET("/:id", productHandler.GetByID)
		products.POST("", productHandler.Create)
		products.PUT("/:id", productHandler.Update)
		products.PATCH("/:id", productHandler.Patch)
		products.DELETE("/:id", productHandler.Delete)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
