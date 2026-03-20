package dto

import (
	"errors"
	"regexp"
	"strings"
)

// RegisterRequest представляет данные для регистрации пользователя
type RegisterRequest struct {
	Email    string `json:"email" binding:"required" format:"email" example:"student@example.com"`
	Password string `json:"password" binding:"required" minLength:"8" example:"StrongPass123"`
	Phone    string `json:"phone,omitempty" example:"+79991234567"`
}

// Validate проверяет корректность данных регистрации
func (r *RegisterRequest) Validate() error {
	// Проверка email
	if !isValidEmail(r.Email) {
		return errors.New("некорректный формат email")
	}

	// Проверка сложности пароля
	if err := validatePassword(r.Password); err != nil {
		return err
	}

	// Проверка телефона (если указан)
	if r.Phone != "" && !isValidPhone(r.Phone) {
		return errors.New("некорректный формат телефона")
	}

	return nil
}

// isValidEmail проверяет формат email
func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, strings.TrimSpace(email))
	return matched
}

// isValidPhone проверяет формат телефона
func isValidPhone(phone string) bool {
	pattern := `^\+?[0-9]{10,15}$`
	matched, _ := regexp.MatchString(pattern, strings.TrimSpace(phone))
	return matched
}

// validatePassword проверяет сложность пароля
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("пароль должен содержать минимум 8 символов")
	}
	if len(password) > 128 {
		return errors.New("пароль не должен превышать 128 символов")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false

	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		}
	}
	// Проверка только обязательных типов символов
	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("пароль должен содержать заглавные и строчные буквы, а также цифры")
	}

	return nil
}
