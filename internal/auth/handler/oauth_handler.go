package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lab2/rest-api/internal/auth/service"
	"github.com/lab2/rest-api/internal/auth/dto"
)

// OAuthHandler обрабатывает OAuth запросы
type OAuthHandler struct {
	oauthService service.OAuthService
}

// NewOAuthHandler создаёт новый экземпляр OAuth хендлера
func NewOAuthHandler(oauthService service.OAuthService) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
	}
}

// InitOAuth инициирует OAuth авторизацию
// @Summary Инициация OAuth авторизации
// @Tags auth
// @Produce json
// @Param provider path string true "Провайдер (yandex)"
// @Success 302 "Редирект на провайдера"
// @Failure 400 {object} AuthErrorResponse
// @Failure 401 {object} AuthErrorResponse
// @Failure 403 {object} AuthErrorResponse
// @Failure 404 {object} AuthErrorResponse
// @Failure 500 {object} AuthErrorResponse
// @Router /auth/oauth/{provider} [get]
func (h *OAuthHandler) InitOAuth(c *gin.Context) {
	provider := c.Param("provider")

	authURL, state, err := h.oauthService.GetAuthorizationURL(provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Сохраняем state в cookie для проверки в callback.
	// Path "/" — чтобы браузер отправил куку при редиректе на /auth/oauth/{provider}/callback.
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)

	c.Redirect(http.StatusFound, authURL)
}

// OAuthCallback обрабатывает callback от OAuth провайдера
// @Summary Обработка OAuth callback
// @Tags auth
// @Produce json
// @Param provider path string true "Провайдер (yandex)"
// @Param code query string true "Код авторизации"
// @Param state query string true "State токен"
// @Success 200 {object} dto.OAuthCallbackResponse "успешный вход через OAuth (JWT в HttpOnly cookies)"
// @Failure 400 {object} AuthErrorResponse
// @Failure 401 {object} AuthErrorResponse
// @Failure 403 {object} AuthErrorResponse
// @Failure 404 {object} AuthErrorResponse
// @Failure 500 {object} AuthErrorResponse
// @Router /auth/oauth/{provider}/callback [get]
func (h *OAuthHandler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	// Проверяем state из cookie
	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState != state {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "невалидный state"})
		return
	}

	// Обрабатываем callback
	user, tokens, err := h.oauthService.HandleCallback(c.Request.Context(), provider, code, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Устанавливаем JWT cookies
	c.SetCookie(
		"access_token",
		tokens.AccessToken,
		int(tokens.AccessExpiresIn.Seconds()),
		"/",
		"",
		false,
		true,
	)

	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		int(tokens.RefreshExpiresIn.Seconds()),
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, dto.OAuthCallbackResponse{
		Message: "успешный вход через OAuth",
		UserID:  user.ID,
		Email:   user.Email,
	})
}
