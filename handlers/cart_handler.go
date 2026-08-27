package handlers

import (
	"checkout-api/apierrors"
	"checkout-api/models"
	"checkout-api/validation"
	"encoding/json"
	"net/http"
	"strconv"
)

func (h *Handler) CreateUserCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierrors.Write(
			w,
			http.StatusMethodNotAllowed,
			apierrors.CodeNotAllowed,
			"Method not allowed",
		)
		return
	}

	var cartReq CreateCartRequest

	err := json.NewDecoder(r.Body).Decode(&cartReq)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"Invalid request body",
		)
		return
	}

	///////////////////
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"missing X-User-ID header",
		)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid X-User-ID header",
		)
		return
	}

	///////////////////

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	items, err := h.store.GetUserCart(r.Context(), userID)
	if len(items) != 0 {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"cart already exists",
		)
		return
	}

	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			err.Error(),
		)
		return
	}

	if err := validation.QuantityProvided(&cartReq.Quantity); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeValidationError,
			err.Error(),
		)
		return
	}

	if err := validation.QuantityIsPositive(cartReq.Quantity); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeBusinessRuleViolation,
			err.Error(),
		)
		return
	}

	if err := validation.ItemID(cartReq.ItemID); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeValidationError,
			err.Error(),
		)
		return
	}

	item, err := h.store.GetItem(r.Context(), cartReq.ItemID)
	if err != nil || item == nil {
		apierrors.Write(
			w,
			http.StatusNotFound,
			apierrors.CodeNotFound,
			"item with that ID doesn't exist",
		)
		return
	}

	if err := validation.QuantityWithinStock(cartReq.Quantity, item.Stock); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeBusinessRuleViolation,
			"quantity is greater than stock",
		)
		return
	}

	cart := models.Cart{
		UserID:   userID,
		ItemID:   cartReq.ItemID,
		Quantity: cartReq.Quantity,
	}
	err = h.store.CreateUserCart(r.Context(), &cart)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"failed to create cart",
		)
		return
	}

	writeJSON(w, http.StatusCreated, cart)
}

func (h *Handler) UpsertCartItem(w http.ResponseWriter, r *http.Request) {

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"missing X-User-ID header",
		)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid X-User-ID header",
		)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"missing item_id",
		)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"item_id must be an integer",
		)
		return
	}

	if err := validation.ItemID(itemID); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeValidationError,
			err.Error(),
		)
		return
	}

	item, err := h.store.GetItem(r.Context(), itemID)
	if err != nil || item == nil {
		apierrors.Write(
			w,
			http.StatusNotFound,
			apierrors.CodeNotFound,
			"item not found with this ID",
		)
		return
	}

	var req UpsertCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid request body",
		)
		return
	}

	if err := validation.QuantityProvided(req.Quantity); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeValidationError,
			err.Error(),
		)
		return
	}

	quantity := *req.Quantity

	if err := validation.QuantityIsPositive(quantity); err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeBusinessRuleViolation,
			err.Error(),
		)
		return
	}
	if err := validation.QuantityWithinStock(quantity, item.Stock); err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeBusinessRuleViolation,
			err.Error(),
		)
		return
	}

	if err := h.store.UpsertCartItem(r.Context(), userID, itemID, quantity); err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"ran into a problem while executing the update",
		)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"missing X-User-ID header",
		)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid X-User-ID header",
		)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"missing item_id",
		)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeInvalidRequest,
			"item_id must be integer",
		)
		return
	}

	if err := validation.ItemID(itemID); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeValidationError,
			err.Error(),
		)
		return
	}

	if err := h.store.RemoveCartItem(r.Context(), userID, itemID); err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"missing X-User-ID header",
		)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid X-User-ID header",
		)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	cart, err := h.store.GetUserCart(r.Context(), userID)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusNotFound,
			apierrors.CodeNotFound,
			err.Error(),
		)
		return
	}

	items := make([]CartItemResponse, 0, len(cart))
	for _, item := range cart {
		items = append(items, CartItemResponse{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Price:       item.Price,
			Stock:       item.Stock,
			CreatedAt:   item.CreatedAt,
			Quantity:    item.Quantity,
		})
	}

	writeJSON(w, http.StatusOK, CartResponse{
		UserID: userID,
		Items:  items,
	})
}

func (h *Handler) DeleteUserCart(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"missing X-User-ID header",
		)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid X-User-ID header",
		)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	err = h.store.DeleteUserCart(r.Context(), userID)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusNotFound,
			apierrors.CodeNotFound,
			err.Error(),
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
