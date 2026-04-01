package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lab2/rest-api/internal/auth/repository"
	"github.com/lab2/rest-api/internal/auth/service"
	"github.com/lab2/rest-api/internal/cache"
)

// AuthMiddleware проверяет access token при каждом запросе к защищённым эндпоинтам.
// Выполняет два уровня проверки:
//  1. Криптографическая валидация подписи и срока действия JWT.
//  2. Проверка по БД — хэш access token должен существовать в активной (не отозванной) сессии.
//     Это гарантирует мгновенную инвалидацию токена после logout.
func AuthMiddleware(jwtService service.JWTService, tokenRepo repository.TokenRepository, cacheSvc cache.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Извлекаем access token из HttpOnly cookie
		tokenString, err := c.Cookie("access_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "токен авторизации не найден",
			})
			return
		}

		// 2. Проверяем подпись JWT и срок действия
		claims, err := jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "невалидный или истёкший токен",
			})
			return
		}

		// 2.1 Проверяем наличие JTI в Redis (если кеш доступен).
		if claims.ID != "" {
			exists, err := cacheSvc.Exists(c.Request.Context(), cache.UserAccessJTIKey(claims.UserID, claims.ID))
			if err == nil && !exists {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "сессия неактивна, выполните вход повторно",
				})
				return
			}
			if err != nil {
				log.Printf("auth middleware: Redis недоступен, продолжаем проверку через БД: %v", err)
			}
		}

		// 3. Хэшируем токен и ищем его в БД.
		// Если сессия отозвана (logout) или не существует — запрос отклоняется.
		tokenHash := hashAccessToken(tokenString)
		session, err := tokenRepo.GetByAccessTokenHash(c.Request.Context(), tokenHash)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "ошибка проверки сессии",
			})
			return
		}
		if session == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "сессия не найдена или завершена, выполните вход повторно",
			})
			return
		}

		// 4. Сохраняем userID в контекст для использования в хендлерах
		c.Set("userID", claims.UserID.String())
		c.Set("userEmail", claims.UserID.String())

		c.Next()
	}
}

// hashAccessToken вычисляет SHA-256 хэш токена для поиска в БД.
func hashAccessToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// CORSMiddleware настраивает заголовки CORS для фронтенда.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
