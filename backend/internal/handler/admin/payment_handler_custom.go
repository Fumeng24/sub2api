package admin

import (
	"strconv"
	"strings"
)

func parseOptionalBoolQuery(raw string) *bool {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return nil
	}
	return &v
}
