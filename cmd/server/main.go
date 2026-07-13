// @title Lab 2–9 REST API
// @version 1.4
// @description REST API на Go (Gin + MongoDB + Redis + MinIO + RabbitMQ) с JWT/OAuth2-аутентификацией, CRUD-ресурсами, файловым хранилищем, асинхронной обработкой событий и Kubernetes-деплоем. ЛР9: health-зонды (/health/live, /health/ready), K8s-манифесты, горизонтальное масштабирование, Redis distributed lock.
// @host localhost:4200
// @BasePath /
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name access_token
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	_ "github.com/lab2/rest-api/docs"
	"github.com/lab2/rest-api/internal/cache"
	categoryhandler "github.com/lab2/rest-api/internal/category/handler"
	categoryrepo "github.com/lab2/rest-api/internal/category/repository"
	categorysvc "github.com/lab2/rest-api/internal/category/service"
	clusterhandler "github.com/lab2/rest-api/internal/cluster/handler"
	clustersource "github.com/lab2/rest-api/internal/cluster/source"
	"github.com/lab2/rest-api/internal/config"
	"github.com/lab2/rest-api/internal/database"
	"github.com/lab2/rest-api/internal/email"
	filehandler "github.com/lab2/rest-api/internal/file/handler"
	filerepo "github.com/lab2/rest-api/internal/file/repository"
	fileservice "github.com/lab2/rest-api/internal/file/service"
	"github.com/lab2/rest-api/internal/health"
	incidenthandler "github.com/lab2/rest-api/internal/incident/handler"
	"github.com/lab2/rest-api/internal/incident/hub"
	incidentrepo "github.com/lab2/rest-api/internal/incident/repository"
	incidentservice "github.com/lab2/rest-api/internal/incident/service"
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
	"github.com/lab2/rest-api/internal/messaging"
	"github.com/lab2/rest-api/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.AppEnv == "development" {
		log.Printf("SMTP (без секретов): %s:%d user=%s from=%s secure=%v auth=%s",
			cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPFrom, cfg.SMTPSecure, cfg.SMTPAuth)
	}

	// ========== MONGODB ==========
	ctx := context.Background()
	appCtx, appCancel := context.WithCancel(ctx)
	defer appCancel()
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
	colIncidents := mongoDB.Collection("incidents")
	colIncidentEvents := mongoDB.Collection("incident_events")

	// ========== REDIS / CACHE ==========
	cacheClient, cacheErr := cache.NewRedisClient(ctx, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if cacheErr != nil {
		log.Printf("Redis недоступен, продолжаем без кеша: %v", cacheErr)
	}
	cacheService := cache.NewService(cacheClient, cfg.CacheEnabled)

	// ========== RabbitMQ + фоновый консьюмер (ЛР8) ==========
	rmqConn, err := messaging.DialAMQP(cfg.AMQPURI())
	if err != nil {
		log.Fatalf("подключение к RabbitMQ: %v", err)
	}
	defer rmqConn.Close()

	rmqPubCh, err := rmqConn.Channel()
	if err != nil {
		log.Fatalf("канал RabbitMQ (publisher): %v", err)
	}
	if err := messaging.DeclareTopology(rmqPubCh, cfg.QueueRegistered); err != nil {
		log.Fatalf("топология RabbitMQ: %v", err)
	}
	eventPublisher := messaging.NewPublisher(rmqPubCh)
	mailSender := email.NewSender(cfg)

	rmqConsCh, err := rmqConn.Channel()
	if err != nil {
		log.Fatalf("канал RabbitMQ (consumer): %v", err)
	}
	instanceID := fmt.Sprintf("%s:%s", getHostname(), uuid.New().String())
	distLock := cache.NewDistributedLock(cacheClient)
	go messaging.RunUserRegisteredConsumer(appCtx, rmqConsCh, cfg.QueueRegistered, cacheService, distLock, instanceID, mailSender, eventPublisher)

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
		eventPublisher,
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
		eventPublisher,
	)
	oauthHandler := authhandler.NewOAuthHandler(oauthService)

	authMW := authmiddleware.AuthMiddlewareWithUsers(jwtService, tokenRepo, userRepo, cacheService)

	// ========== КАТЕГОРИИ И ПРОДУКТЫ ==========
	categoryRepo := categoryrepo.NewCategoryRepository(colCategories)
	productRepo := productrepo.NewProductRepository(colProducts, colCategories)
	fileSvc := fileservice.NewService(fileRepo, storageService, cacheService, cfg.CacheTTLDefault, cfg.MinIOBucket, cfg.MaxFileSize)
	categorySvc := categorysvc.NewCategoryService(categoryRepo, productRepo, cacheService, cfg.CacheTTLDefault)
	productSvc := productsvc.NewProductService(productRepo, categoryRepo, cacheService, cfg.CacheTTLDefault)
	categoryHandler := categoryhandler.NewCategoryHandler(categorySvc)
	productHandler := producthandler.NewProductHandler(productSvc)
	fileHandler := filehandler.NewHandler(fileSvc, storageService)
	cluster, scenarios, err := clustersource.NewFactory(cfg.ClusterSource, 3800*time.Millisecond)
	if err != nil {
		log.Fatalf("init cluster source: %v", err)
	}
	incidentHub := hub.New()
	defer incidentHub.Close()
	incidentRepo := incidentrepo.NewMongoRepository(colIncidents, colIncidentEvents)
	incidentSvc := incidentservice.New(incidentRepo, cluster, incidentHub, fileRepo, storageService, cfg.MinIOBucket, cfg.MaxFileSize, userRepo)
	incidentHandler := incidenthandler.New(incidentSvc, userRepo, storageService)

	// ========== РОУТЕР ==========
	r := gin.New()
	r.MaxMultipartMemory = cfg.MaxFileSize
	r.Use(gin.Recovery(), middleware.Recovery(), middleware.CORS(cfg.CORSAllowedOrigins))
	clusterhandler.New(cluster, scenarios).RegisterRoutes(r)
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

	r.GET("/health/live", health.LiveHandler())
	r.GET("/health/ready", health.ReadyHandler(mongoDB, cacheClient, rmqConn))
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

	incidentAPI := r.Group("")
	incidentAPI.Use(authMW)
	incidentHandler.RegisterRoutes(incidentAPI)

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
		files.GET("", fileHandler.List)
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
	log.Printf("server listening on %s (instanceID=%s)", addr, instanceID)
	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return appCtx
		},
		// Do not set WriteTimeout: SSE connections are intentionally long-lived.
	}
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case serveErr := <-serverErr:
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Fatalf("serve: %v", serveErr)
		}
	case sig := <-signals:
		log.Printf("received %s, shutting down", sig)
		appCancel()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown: %v", err)
		}
		cancel()
	}
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
