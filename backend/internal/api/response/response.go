package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type APIError struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
}

type PaginatedResponse[T any] struct {
	Success bool `json:"success"`
	Data    []T  `json:"data"`
	Meta    Meta `json:"meta"`
}

type Meta struct {
	Total  int `json:"total"`
	Page   int `json:"page"`
	Limit  int `json:"limit"`
	Pages  int `json:"pages"`
}

func Success[T any](c echo.Context, data T) error {
	return c.JSON(http.StatusOK, APIResponse[T]{
		Success: true,
		Data:    data,
	})
}

func Created[T any](c echo.Context, data T) error {
	return c.JSON(http.StatusCreated, APIResponse[T]{
		Success: true,
		Data:    data,
	})
}

func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func ErrorResponse(c echo.Context, status int, message string) error {
	return c.JSON(status, APIError{
		Success: false,
		Error:   message,
	})
}

func ErrorWithCode(c echo.Context, status int, message, code string) error {
	return c.JSON(status, APIError{
		Success: false,
		Error:   message,
		Code:    code,
	})
}

func BadRequest(c echo.Context, message string) error {
	return ErrorResponse(c, http.StatusBadRequest, message)
}

func Unauthorized(c echo.Context, message string) error {
	return ErrorResponse(c, http.StatusUnauthorized, message)
}

func Forbidden(c echo.Context, message string) error {
	return ErrorResponse(c, http.StatusForbidden, message)
}

func NotFound(c echo.Context, message string) error {
	return ErrorResponse(c, http.StatusNotFound, message)
}

func InternalError(c echo.Context, message string) error {
	return ErrorResponse(c, http.StatusInternalServerError, message)
}

func ServiceUnavailable(c echo.Context, message string) error {
	return ErrorResponse(c, http.StatusServiceUnavailable, message)
}

func Paginated[T any](c echo.Context, data []T, total, page, limit int) error {
	pages := total / limit
	if total%limit > 0 {
		pages++
	}
	return c.JSON(http.StatusOK, PaginatedResponse[T]{
		Success: true,
		Data:    data,
		Meta: Meta{
			Total: total,
			Page:  page,
			Limit: limit,
			Pages: pages,
		},
	})
}
