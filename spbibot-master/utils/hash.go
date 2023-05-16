package utils

import (
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// HashString method creates a hash string based on string and cost parameters
func HashString(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		zap.L().Error("couldn't make hash", zap.Error(err))

		return "", err
	}

	return string(hash), nil
}
