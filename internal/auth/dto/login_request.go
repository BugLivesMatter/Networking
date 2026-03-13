package dto

import "errors"

// LoginRequest представляет данные для входа пользователя
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Validate проверяет корректность данных входа
func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email обязателен")
	}
	if r.Password == "" {
		return errors.New("пароль обязателен")
	}
	return nil
}
