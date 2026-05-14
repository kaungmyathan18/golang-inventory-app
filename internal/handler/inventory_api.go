package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kaungmyathan18/golang-inventory-app/internal/apiresponse"
	"github.com/kaungmyathan18/golang-inventory-app/internal/observability"
	"github.com/kaungmyathan18/golang-inventory-app/internal/repository"
	"github.com/kaungmyathan18/golang-inventory-app/internal/service"
	"github.com/kaungmyathan18/golang-inventory-app/internal/validation"
	"go.uber.org/zap"
)

type InventoryAPIHandler struct {
	svc     *service.InventoryService
	repo    *repository.ProductRepository
	log     *zap.Logger
	metrics *observability.Metrics
}

func NewInventoryAPIHandler(svc *service.InventoryService, log *zap.Logger, m *observability.Metrics, repo *repository.ProductRepository) *InventoryAPIHandler {
	return &InventoryAPIHandler{svc: svc, repo: repo, log: log, metrics: m}
}

type createInventoryReq struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=0"`
}

func (h *InventoryAPIHandler) CreateInventory(w http.ResponseWriter, r *http.Request) {
	var req createInventoryReq
	if err := validation.DecodeJSON(r, &req); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	i, err := h.svc.CreateInventory(r.Context(), req.ProductID, req.Quantity)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusCreated, i, nil, nil)
	return
}

func (h *InventoryAPIHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	i, err := h.svc.GetInventory(r.Context(), id)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, i, nil, nil)
	return
}

func (h *InventoryAPIHandler) ListInventories(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePageLimit(r)
	lq := struct {
		Page  int `validate:"gte=1"`
		Limit int `validate:"gte=1,lte=100"`
	}{Page: page, Limit: limit}
	if err := validation.V.Struct(lq); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	inventories, total, err := h.svc.ListInventoriesPaged(r.Context(), page, limit)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	offset := (page - 1) * limit
	hasMore := int64(offset+len(inventories)) < total
	pagination := &apiresponse.PaginationMeta{
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasMore: hasMore,
	}
	links := apiresponse.PageLinks(r, page, limit, len(inventories) == limit)
	apiresponse.WriteJSON(w, r, http.StatusOK, inventories, pagination, links)
	return
}

func (h *InventoryAPIHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	i, err := h.svc.GetInventory(r.Context(), id)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	if i == nil {
		apiresponse.WriteProblem(w, r, http.StatusNotFound,
			apiresponse.ProblemTypeURI(r, "not-found"),
			"Not Found",
			"No inventory exists for the given id.",
			nil,
		)
		return
	}
	if i.ProductID == "" {
		apiresponse.WriteProblem(w, r, http.StatusNotFound,
			apiresponse.ProblemTypeURI(r, "not-found"),
			"Not Found",
			"No product exists for the given inventory id.",
			nil,
		)
		return
	}
	_, err = h.repo.Get(r.Context(), i.ProductID)
	if err != nil {
		apiresponse.WriteProblem(w, r, http.StatusNotFound,
			apiresponse.ProblemTypeURI(r, "not-found"),
			"Not Found",
			"No product exists for the given inventory id.",
			nil,
		)
		return
	}
	var req updateInventoryReq
	if err := validation.DecodeJSON(r, &req); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	i, err = h.svc.UpdateInventory(r.Context(), id, req.ProductID, req.Quantity)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, i, nil, nil)
	return
}

type updateInventoryReq struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=0"`
}

func (h *InventoryAPIHandler) DeleteInventory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	err := h.svc.DeleteInventory(r.Context(), id)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, nil, nil, nil)
	return
}
