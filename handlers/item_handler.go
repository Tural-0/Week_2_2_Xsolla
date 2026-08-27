package handlers

import (
	"checkout-api/apierrors"
	"checkout-api/models"
	"checkout-api/pagination"
	"checkout-api/validation"
	"fmt"
	"net/http"
	"strconv"
)

// GetItems handles GET /items — returns all available items.
func (h *Handler) GetItems(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	// Cursor pagination

	if query.Get("cursor") != "" {
		params, err := pagination.ParseCursor(r)
		if err != nil {
			apierrors.Write(
				w,
				http.StatusBadRequest,
				apierrors.CodeInvalidRequest,
				err.Error(),
			)
			return
		}

		items, err := h.store.GetItemsCursor(
			r.Context(),
			params.Limit,
			params.Cursor,
		)
		if err != nil {
			apierrors.Write(
				w,
				http.StatusInternalServerError,
				apierrors.CodeInternal,
				err.Error(),
				// "internal server error at GetItemsCursor",
			)
			return
		}

		var nextCursor *int

		if len(items) > 0 {
			lastID := items[len(items)-1].ID
			nextCursor = &lastID
		}

		response := CursorResponse[models.Item]{
			Data: items,
			Pagination: CursorMeta{
				Limit:      params.Limit,
				NextCursor: nextCursor,
			},
		}

		writeJSON(w, http.StatusOK, response)
		return
	}

	// Offset pagination

	params, err := pagination.ParseOffset(r)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			err.Error(),
		)
		return
	}

	items, err := h.store.GetItemsOffset(
		r.Context(),
		params.Limit,
		params.Offset,
	)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	response := OffsetResponse[models.Item]{
		Data: items,
		Pagination: OffsetMeta{
			Limit:  params.Limit,
			Offset: params.Offset,
		},
	}

	writeJSON(w, http.StatusOK, response)

	// items, err = h.store.GetItems(r.Context())
	// if err != nil {
	// 	apierrors.Write(
	// 		w,
	// 		http.StatusInternalServerError,
	// 		apierrors.CodeInternal,
	// 		err.Error(),
	// 	)
	// 	return
	// }

	// writeJSON(w, http.StatusOK, items)
}

// GetItemByID handles GET /items/{id} — returns a single i`tem.
func (h *Handler) GetItemByID(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeInvalidRequest,
			"URL doesn't contain item ID",
		)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
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
	if err != nil {
		fmt.Printf("error: %v", err)
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	if item == nil {
		apierrors.Write(
			w,
			http.StatusNotFound,
			apierrors.CodeNotFound,
			"item not found with this ID",
		)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) GetItemQuantityByID(w http.ResponseWriter, r *http.Request) {
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

	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeInvalidRequest,
			"URL doesn't contain item_id inside",
		)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
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

	quantity, err := h.store.GetItemQuantityByID(r.Context(), userID, itemID)

	if quantity <= 0 && err == nil {
		apierrors.Write(
			w,
			http.StatusNotFound,
			apierrors.CodeNotFound,
			"this item doesn't exist",
		)
		return
	}

	if err != nil {
		fmt.Printf("error: %v", err)
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusOK, quantity)
}
