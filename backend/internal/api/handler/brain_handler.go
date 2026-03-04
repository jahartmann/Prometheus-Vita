package handler

import (
	"errors"

	apiPkg "github.com/antigravity/prometheus/internal/api/response"
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/repository"
	"github.com/antigravity/prometheus/internal/service/brain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type BrainHandler struct {
	service *brain.Service
}

func NewBrainHandler(service *brain.Service) *BrainHandler {
	return &BrainHandler{service: service}
}

// List handles GET /api/v1/brain.
func (h *BrainHandler) List(c echo.Context) error {
	entries, err := h.service.List(c.Request().Context())
	if err != nil {
		return apiPkg.InternalError(c, "failed to list brain entries")
	}
	return apiPkg.Success(c, entries)
}

// Create handles POST /api/v1/brain.
func (h *BrainHandler) Create(c echo.Context) error {
	var req model.CreateBrainEntryRequest
	if err := c.Bind(&req); err != nil {
		return apiPkg.BadRequest(c, "invalid request body")
	}

	if req.Category == "" || req.Subject == "" || req.Content == "" {
		return apiPkg.BadRequest(c, "category, subject and content are required")
	}

	entry, err := h.service.Create(c.Request().Context(), req, nil)
	if err != nil {
		return apiPkg.InternalError(c, "failed to create brain entry")
	}

	return apiPkg.Created(c, entry)
}

// Delete handles DELETE /api/v1/brain/:id.
func (h *BrainHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apiPkg.BadRequest(c, "invalid id")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apiPkg.NotFound(c, "brain entry not found")
		}
		return apiPkg.InternalError(c, "failed to delete brain entry")
	}

	return apiPkg.NoContent(c)
}

// Search handles GET /api/v1/brain/search?q=.
func (h *BrainHandler) Search(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return apiPkg.BadRequest(c, "query parameter 'q' is required")
	}

	entries, err := h.service.Search(c.Request().Context(), query)
	if err != nil {
		return apiPkg.InternalError(c, "failed to search brain entries")
	}

	return apiPkg.Success(c, entries)
}
