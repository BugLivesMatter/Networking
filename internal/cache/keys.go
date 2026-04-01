package cache

import (
	"fmt"

	"github.com/google/uuid"
)

const appPrefix = "wp"

func ProductsListKey(page, limit int, categoryID string) string {
	if categoryID == "" {
		return fmt.Sprintf("%s:products:list:page:%d:limit:%d", appPrefix, page, limit)
	}
	return fmt.Sprintf("%s:products:list:page:%d:limit:%d:category:%s", appPrefix, page, limit, categoryID)
}

func ProductsListPattern() string {
	return appPrefix + ":products:list:*"
}

func CategoriesListKey(page, limit int) string {
	return fmt.Sprintf("%s:categories:list:page:%d:limit:%d", appPrefix, page, limit)
}

func CategoriesListPattern() string {
	return appPrefix + ":categories:list:*"
}

func UserProfileKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s:users:profile:%s", appPrefix, userID.String())
}

func UserAccessJTIKey(userID uuid.UUID, jti string) string {
	return fmt.Sprintf("%s:auth:user:%s:access:%s", appPrefix, userID.String(), jti)
}

func UserAccessJTIPattern(userID uuid.UUID) string {
	return fmt.Sprintf("%s:auth:user:%s:access:*", appPrefix, userID.String())
}
