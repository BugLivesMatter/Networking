package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lab2/rest-api/internal/auth/dto"
	"github.com/lab2/rest-api/internal/auth/service"
)

// PasswordHandler обрабатывает запросы сброса пароля
type PasswordHandler struct {
	authService service.AuthService
}

// NewPasswordHandler создаёт новый экземпляр хендлера
func NewPasswordHandler(authService service.AuthService) *PasswordHandler {
	return &PasswordHandler{
		authService: authService,
	}
}

// ForgotPassword обрабатывает POST /auth/forgot-password
// @Summary Запрос сброса пароля
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "Email пользователя"
// @Success 200 {object} dto.ForgotPasswordResponse "сообщение о выполнении запроса"
// @Failure 400 {object} AuthErrorResponse
// @Failure 401 {object} AuthErrorResponse
// @Failure 403 {object} AuthErrorResponse
// @Failure 404 {object} AuthErrorResponse
// @Failure 500 {object} AuthErrorResponse
// @Router /auth/forgot-password [post]
func (h *PasswordHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные запроса"})
		return
	}

	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка при обработке запроса"})
		return
	}

	c.JSON(http.StatusOK, dto.ForgotPasswordResponse{
		Message: "если пользователь существует, письмо для сброса пароля отправлено",
	})
}

// ResetPassword обрабатывает POST /auth/reset-password
// @Summary Установка нового пароля по токену
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "Токен и новый пароль"
// @Success 200 {object} dto.ResetPasswordResponse "пароль успешно изменён"
// @Failure 400 {object} AuthErrorResponse
// @Failure 401 {object} AuthErrorResponse
// @Failure 403 {object} AuthErrorResponse
// @Failure 404 {object} AuthErrorResponse
// @Failure 500 {object} AuthErrorResponse
// @Router /auth/reset-password [post]
func (h *PasswordHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "некорректные данные запроса"})
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.ResetPasswordResponse{
		Message: "пароль успешно изменён",
	})
}
