package dto

// ForgotPasswordRequest содержит запрос на сброс пароля
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email" format:"email" example:"student@example.com"`
}

// ResetPasswordRequest содержит запрос на установку нового пароля
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required" example:"9f6c2b7e-7f6a-4ad0-9e0a-0b8c3d4e5f60"`
	NewPassword string `json:"new_password" binding:"required,min=8" minLength:"8" example:"NewStrongPass123"`
}
