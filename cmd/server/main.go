// @title Lab 2–6 REST API
// @version 1.2
// @description REST API: категории и продукты (CRUD), JWT + OAuth2, Redis (кеш списков и профиля, JTI access в Redis), MongoDB вместо PostgreSQL. Health-эндпоинты для мониторинга Redis и диагностики латентности MongoDB vs кеша.
// @host localhost:4200
// @BasePath /
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name access_token
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lab2/rest-api/docs"
	"github.com/lab2/rest-api/internal/cache"
	categoryhandler "github.com/lab2/rest-api/internal/category/handler"
	categoryrepo "github.com/lab2/rest-api/internal/category/repository"
	categorysvc "github.com/lab2/rest-api/internal/category/service"
	"github.com/lab2/rest-api/internal/config"
	"github.com/lab2/rest-api/internal/database"
	filehandler "github.com/lab2/rest-api/internal/file/handler"
	filerepo "github.com/lab2/rest-api/internal/file/repository"
	fileservice "github.com/lab2/rest-api/internal/file/service"
	"github.com/lab2/rest-api/internal/health"
	"github.com/lab2/rest-api/internal/middleware"
	producthandler "github.com/lab2/rest-api/internal/product/handler"
	productrepo "github.com/lab2/rest-api/internal/product/repository"
	productsvc "github.com/lab2/rest-api/internal/product/service"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Auth модуль (с алиасами!)
	authhandler "github.com/lab2/rest-api/internal/auth/handler"
	authmiddleware "github.com/lab2/rest-api/internal/auth/middleware"
	authrepo "github.com/lab2/rest-api/internal/auth/repository"
	authservice "github.com/lab2/rest-api/internal/auth/service"
	"github.com/lab2/rest-api/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// ========== MONGODB ==========
	ctx := context.Background()
	mongoClient, err := database.Connect(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatalf("connect mongodb: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	mongoDB := mongoClient.Database(cfg.MongoDBName)

	if err := database.EnsureIndexes(ctx, mongoDB); err != nil {
		log.Fatalf("ensure indexes: %v", err)
	}
	log.Println("MongoDB индексы успешно созданы/проверены")

	// ========== КОЛЛЕКЦИИ ==========
	colUsers := mongoDB.Collection("users")
	colTokens := mongoDB.Collection("refresh_tokens")
	colPassReset := mongoDB.Collection("password_reset_tokens")
	colCategories := mongoDB.Collection("categories")
	colProducts := mongoDB.Collection("products")
	colFiles := mongoDB.Collection("files")

	// ========== REDIS / CACHE ==========
	cacheClient, cacheErr := cache.NewRedisClient(ctx, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if cacheErr != nil {
		log.Printf("Redis недоступен, продолжаем без кеша: %v", cacheErr)
	}
	cacheService := cache.NewService(cacheClient, cfg.CacheEnabled)

	storageService, err := storage.NewMinIOService(cfg)
	if err != nil {
		log.Fatalf("init minio service: %v", err)
	}

	// ========== AUTH МОДУЛЬ ==========
	userRepo := authrepo.NewUserRepository(colUsers)
	tokenRepo := authrepo.NewTokenRepository(colTokens)
	fileRepo := filerepo.NewFileRepository(colFiles)

	passwordService := authservice.NewPasswordService()

	accessDur, _ := time.ParseDuration(cfg.JWTAccessExpiration)
	refreshDur, _ := time.ParseDuration(cfg.JWTRefreshExpiration)
	jwtService := authservice.NewJWTService(
		cfg.JWTAccessSecret,
		cfg.JWTRefreshSecret,
		accessDur,
		refreshDur,
	)

	passwordResetRepo := authrepo.NewPasswordResetRepository(colPassReset)
	authService := authservice.NewAuthService(
		userRepo,
		tokenRepo,
		passwordService,
		jwtService,
		passwordResetRepo,
		fileRepo,
		cacheService,
		cfg.CacheTTLDefault,
	)

	authHandler := authhandler.NewAuthHandler(authService)
	passwordHandler := authhandler.NewPasswordHandler(authService)

	// ========== OAUTH ==========
	oauthConfig := &authservice.OAuthConfig{
		YandexClientID:     cfg.YandexClientID,
		YandexClientSecret: cfg.YandexClientSecret,
		YandexRedirectURI:  cfg.YandexCallbackURL,
	}
	oauthService := authservice.NewOAuthService(
		userRepo,
		tokenRepo,
		passwordService,
		jwtService,
		cacheService,
		cfg.CacheTTLDefault,
		oauthConfig,
	)
	oauthHandler := authhandler.NewOAuthHandler(oauthService)

	authMW := authmiddleware.AuthMiddleware(jwtService, tokenRepo, cacheService)

	// ========== КАТЕГОРИИ И ПРОДУКТЫ ==========
	categoryRepo := categoryrepo.NewCategoryRepository(colCategories)
	productRepo := productrepo.NewProductRepository(colProducts, colCategories)
	fileSvc := fileservice.NewService(fileRepo, storageService, cacheService, cfg.CacheTTLDefault, cfg.MinIOBucket, cfg.MaxFileSize)
	categorySvc := categorysvc.NewCategoryService(categoryRepo, productRepo, cacheService, cfg.CacheTTLDefault)
	productSvc := productsvc.NewProductService(productRepo, categoryRepo, cacheService, cfg.CacheTTLDefault)
	categoryHandler := categoryhandler.NewCategoryHandler(categorySvc)
	productHandler := producthandler.NewProductHandler(productSvc)
	fileHandler := filehandler.NewHandler(fileSvc, storageService)

	// ========== РОУТЕР ==========
	r := gin.New()
	r.Use(gin.Recovery(), middleware.Recovery())
	if cfg.AppEnv != "production" {
		swaggerHandler := ginSwagger.WrapHandler(
			swaggerfiles.Handler,
			ginSwagger.PersistAuthorization(true),
		)
		r.GET("/api/docs/*any", func(c *gin.Context) {
			if c.Param("any") == "/swagger-initializer.js" {
				c.Header("Content-Type", "application/javascript")
				c.String(http.StatusOK, `window.onload = function() {
  const ui = SwaggerUIBundle({
    url: "doc.json",
    dom_id: '#swagger-ui',
    validatorUrl: null,
    withCredentials: true,
    requestInterceptor: (request) => {
      request.credentials = 'include';
      return request;
    },
    oauth2RedirectUrl: window.location.origin + window.location.pathname.replace('swagger-initializer.js', '') + 'oauth2-redirect.html',
    persistAuthorization: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: "StandaloneLayout",
    docExpansion: "list",
    deepLinking: true,
    defaultModelsExpandDepth: 1
  })
  window.ui = ui
}`)
				return
			}
			swaggerHandler(c)
		})
	}

	r.GET("/health/redis", cache.StatusHandler(cacheService))
	r.GET("/health/diagnosis", health.DiagnosisHandler(mongoDB, cacheClient, categoryRepo, cacheService, cfg.CacheTTLDefault))

	// ========== PUBLIC ROUTES ==========
	publicAuth := r.Group("/auth")
	{
		publicAuth.POST("/register", authHandler.Register)
		publicAuth.POST("/login", authHandler.Login)
		publicAuth.POST("/refresh", authHandler.Refresh)
		publicAuth.POST("/forgot-password", passwordHandler.ForgotPassword)
		publicAuth.POST("/reset-password", passwordHandler.ResetPassword)
		publicAuth.GET("/oauth/:provider", oauthHandler.InitOAuth)
		publicAuth.GET("/oauth/:provider/callback", oauthHandler.OAuthCallback)
	}

	// ========== PROTECTED ROUTES ==========
	protectedAuth := r.Group("/auth")
	protectedAuth.Use(authMW)
	{
		protectedAuth.GET("/whoami", authHandler.WhoAmI)
		protectedAuth.POST("/logout", authHandler.Logout)
		protectedAuth.POST("/logout-all", authHandler.LogoutAll)
	}

	categories := r.Group("/categories")
	categories.Use(authMW)
	{
		categories.GET("", categoryHandler.List)
		categories.GET("/:id", categoryHandler.GetByID)
		categories.POST("", categoryHandler.Create)
		categories.PUT("/:id", categoryHandler.Update)
		categories.PATCH("/:id", categoryHandler.Patch)
		categories.DELETE("/:id", categoryHandler.Delete)
	}

	products := r.Group("/products")
	products.Use(authMW)
	{
		products.GET("", productHandler.List)
		products.GET("/:id", productHandler.GetByID)
		products.POST("", productHandler.Create)
		products.PUT("/:id", productHandler.Update)
		products.PATCH("/:id", productHandler.Patch)
		products.DELETE("/:id", productHandler.Delete)
	}

	files := r.Group("/files")
	files.Use(authMW)
	{
		files.POST("", fileHandler.Upload)
		files.GET("/:fileId", fileHandler.Download)
		files.DELETE("/:fileId", fileHandler.Delete)
	}

	profile := r.Group("/profile")
	profile.Use(authMW)
	{
		profile.GET("", authHandler.GetProfile)
		profile.POST("", authHandler.UpdateProfile)
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
