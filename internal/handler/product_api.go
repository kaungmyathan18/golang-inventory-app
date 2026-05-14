package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kaungmyathan18/golang-inventory-app/internal/apiresponse"
	"github.com/kaungmyathan18/golang-inventory-app/internal/observability"
	"github.com/kaungmyathan18/golang-inventory-app/internal/repository"
	"github.com/kaungmyathan18/golang-inventory-app/internal/service"
	"github.com/kaungmyathan18/golang-inventory-app/internal/validation"
	"go.uber.org/zap"
)

type ProductAPIHandler struct {
	svc     *service.ProductService
	log     *zap.Logger
	metrics *observability.Metrics
}

func NewProductAPIHandler(svc *service.ProductService, log *zap.Logger, m *observability.Metrics) *ProductAPIHandler {
	return &ProductAPIHandler{svc: svc, log: log, metrics: m}
}

type createProductReq struct {
	Name        string  `json:"name" validate:"required,min=1,max=200"`
	Description string  `json:"description" validate:"required,min=1,max=200"`
	Price       float64 `json:"price" validate:"required,min=0"`
	CategoryID  string  `json:"category_id" validate:"required,uuid"`
}

func (h *ProductAPIHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductReq
	if err := validation.DecodeJSON(r, &req); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	p, err := h.svc.CreateProduct(r.Context(), req.Name, req.Description, req.Price, req.CategoryID)
	if err != nil {
		apiresponse.WriteInternalError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusCreated, p, nil, nil)
}

func (h *ProductAPIHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	p, err := h.svc.GetProduct(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			apiresponse.WriteProblem(w, r, http.StatusNotFound,
				apiresponse.ProblemTypeURI(r, "not-found"),
				"Not Found",
				"No product exists for the given id.",
				nil,
			)
			return
		}
		apiresponse.WriteInternalError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, p, nil, nil)
}

func (h *ProductAPIHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePageLimit(r)
	lq := struct {
		Page  int `validate:"gte=1"`
		Limit int `validate:"gte=1,lte=100"`
	}{Page: page, Limit: limit}
	if err := validation.V.Struct(lq); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	products, total, err := h.svc.ListProductsPaged(r.Context(), page, limit)
	if err != nil {
		apiresponse.WriteInternalError(w, r, err)
		return
	}
	offset := (page - 1) * limit
	hasMore := int64(offset+len(products)) < total
	pagination := &apiresponse.PaginationMeta{
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasMore: hasMore,
	}
	links := apiresponse.PageLinks(r, page, limit, hasMore)
	apiresponse.WriteJSON(w, r, http.StatusOK, products, pagination, links)
}

func (h *ProductAPIHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	var req createProductReq
	if err := validation.DecodeJSON(r, &req); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	p, err := h.svc.UpdateProduct(r.Context(), id, req.Name, req.Description, req.Price, req.CategoryID)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, p, nil, nil)
}

func (h *ProductAPIHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := validation.Var(id, "required,uuid"); err != nil {
		validation.WriteError(w, r, err)
		return
	}
	err := h.svc.DeleteProduct(r.Context(), id)
	if err != nil {
		validation.WriteError(w, r, err)
		return
	}
	apiresponse.WriteJSON(w, r, http.StatusOK, nil, nil, nil)
}
