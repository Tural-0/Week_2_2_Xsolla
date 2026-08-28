package handlers

import (
	"checkout-api/apierrors"
	"checkout-api/models"
	"checkout-api/pagination"
	"checkout-api/validation"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// CreateOrder   godoc
// @Summary      Places the order
// @Description  Places an order according to the request details
// @Tags         Order
// @Produce      json
// @Param        X-User-ID  		header  int     			true  "The user's ID"
// @Param        Idempotency-Key  	header  string     			true  "The idem key"
// @Param		 reqBody			body	CreateOrderRequest	true  "The order details"
// @Success      200  {array}  	byte
// @Failure      400  {object}  apierrors.ErrorDetail
// @Failure      402  {object}  apierrors.ErrorDetail
// @Router       /orders [post]
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
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

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"Idempotency-Key header is required",
		)
		return
	}

	if record, exists := h.idempotencyCache[idempotencyKey]; exists {
		if time.Now().Before(record.Expiry) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			w.Write(record.Response)
			return
		}
		delete(h.idempotencyCache, idempotencyKey)
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid request body",
		)
		return
	}

	if err := validation.NonEmptyCart(req.LineItems); err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeBusinessRuleViolation,
			err.Error(),
		)
		return
	}

	items := make([]models.LineItem, 0, len(req.LineItems))
	for _, i := range req.LineItems {
		items = append(items, models.LineItem{
			ItemID:   i.ItemID,
			Quantity: i.Quantity,
			Price:    i.Price,
		})
	}

	order, err := h.store.CreateOrder(r.Context(), userID, items, req.Total, "pending")
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	paymentResult := mockProcessPayment(req.Total)

	status := "paid"
	if !paymentResult.Success {
		status = "failed"
	}

	if err := h.store.UpdateOrderStatus(r.Context(), order.ID, status); err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}
	order.Status = status

	responseData := map[string]any{
		"order":   order,
		"payment": paymentResult,
	}

	statusCode := http.StatusCreated
	if !paymentResult.Success {
		statusCode = http.StatusPaymentRequired
	}

	responseBody, _ := json.Marshal(responseData)
	h.idempotencyCache[idempotencyKey] = &IdempotencyRecord{
		Response:   responseBody,
		StatusCode: statusCode,
		Expiry:     time.Now().Add(24 * time.Hour),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(responseBody)
}

// GetUserOrders   godoc
// @Summary      Gets the orders of the user
// @Description  Gets the orders of the given user
// @Tags         Order
// @Produce      json
// @Param        X-User-ID  	header 	int     true  "The user's ID"
// @Param        limit  		query  	int     false  "Number of items to return"
// @Param        offset 		query  	int     false  "Number of items to skip"
// @Param        cursor 		query  	int     false  "Cursor for cursor-based pagination"
// @Success      200  {object}  OffsetResponse[models.Order]
// @Failure      400  {object}  apierrors.ErrorDetail
// @Router       /user/orders [get]
func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

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

		orders, err := h.store.GetUserOrdersCursor(
			r.Context(),
			userID,
			params.Limit,
			params.Cursor,
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

		var nextCursor *int

		if len(orders) > 0 {
			lastID := orders[len(orders)-1].ID
			nextCursor = &lastID
		}

		response := CursorResponse[models.Order]{
			Data: orders,
			Pagination: CursorMeta{
				Limit:      params.Limit,
				NextCursor: nextCursor,
			},
		}

		writeJSON(w, http.StatusOK, response)
		return
	}

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

	orders, err := h.store.GetUserOrdersOffset(
		r.Context(),
		userID,
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

	response := OffsetResponse[models.Order]{
		Data: orders,
		Pagination: OffsetMeta{
			Limit:  params.Limit,
			Offset: params.Offset,
		},
	}

	writeJSON(w, http.StatusOK, response)

	// orders, err := h.store.GetUserOrders(r.Context(), userID)
	// if err != nil {
	// 	apierrors.Write(
	// 		w,
	// 		http.StatusInternalServerError,
	// 		apierrors.CodeInternal,
	// 		err.Error(),
	// 	)
	// 	return
	// }

	// ordersResponse := make([]OrderResponse, 0, len(orders))
	// for _, order := range orders {
	// 	items := make([]LinteItemResponse, 0, len(order.Items))

	// 	for _, item := range order.Items {
	// 		items = append(items, LinteItemResponse{
	// 			ItemID:   item.ItemID,
	// 			Quantity: item.Quantity,
	// 			Price:    item.Price,
	// 		})
	// 	}

	// 	ordersResponse = append(ordersResponse, OrderResponse{
	// 		ID:     order.ID,
	// 		UserID: order.UserID,
	// 		Items:  items,
	// 		Total:  order.Total,
	// 		Status: order.Status,
	// 	})
	// }

	// writeJSON(w, http.StatusOK, orders)
}
