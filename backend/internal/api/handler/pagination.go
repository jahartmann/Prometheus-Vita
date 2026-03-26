package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

const (
	DefaultLimit  = 50
	MaxLimit      = 500
	DefaultOffset = 0
)

// ParsePagination extracts and validates limit/offset from query params.
func ParsePagination(c echo.Context) (limit, offset int) {
	limit = DefaultLimit
	offset = DefaultOffset

	if l := c.QueryParam("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	if o := c.QueryParam("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	return limit, offset
}
