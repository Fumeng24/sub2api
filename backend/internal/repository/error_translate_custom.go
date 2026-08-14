package repository

import (
	"errors"
	"strings"

	"github.com/lib/pq"
)

func isUndefinedTableError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01"
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "relation ") && strings.Contains(message, " does not exist")
}
