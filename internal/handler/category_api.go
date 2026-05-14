package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kaungmyathan18/golang-inventory-app/internal/apiresponse"
	"github.com/kaungmyathan18/golang-inventory-app/internal/observability"
	"github.com/kaungmyathan18/golang-inventory-app/internal/service"
	"github.com/kaungmyathan18/golang-inventory-app/internal/validation"
	"go.uber.org/zap"
)

type CategoryAPIHandler struct {
	svc     *service.CategoryService
	log     *zap.Logger
	metrics *observability.Metrics
}

func NewCategoryAPIHandler(svc *service.CategoryService, log *zap.Logger, m *observability.Metrics) *CategoryAPIHandler {
	return &CategoryAPIHandler{svc: svc, log: log, metrics: m}
}

type createCategoryReq struct {
	Name string `json:"name" validate:"required,min=1,max=200"`
}

func (h *CategoryAPIHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req createCategoryReq
	if err := validation.DecodeJSON(r, &req); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	c, err := h.svc.CreateCategory(r.Context(), req.Name)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusCreated, c, nil, nil)
}

func (h *CategoryAPIHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	c, err := h.svc.GetCategory(r.Context(), id)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, c, nil, nil)
}

func (h *CategoryAPIHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePageLimit(r)
	lq := struct {
		Page  int `validate:"gte=1"`
		Limit int `validate:"gte=1,lte=100"`
	}{Page: page, Limit: limit}
	if err := validation.V.Struct(lq); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	categories, err := h.svc.ListCategories(r.Context())
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	links := apiresponse.PageLinks(r, page, limit, len(categories) == limit)
	apiresponse.WriteJSON(w, r, http.StatusOK, categories, nil, links)
}

func (h *CategoryAPIHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	var req createCategoryReq
	if err := validation.DecodeJSON(r, &req); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	c, err := h.svc.UpdateCategory(r.Context(), id, req.Name)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, c, nil, nil)
}

func (h *CategoryAPIHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	err := h.svc.DeleteCategory(r.Context(), id)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, nil, nil, nil)
}
