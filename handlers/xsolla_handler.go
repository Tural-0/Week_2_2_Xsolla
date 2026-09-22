package handlers

import (
	"bytes"
	"checkout-api/apierrors"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const xsollaNotebookSKU = "NotebookA5"

type XsollaPaymentRequest struct {
	Country string `json:"country"`
}

type XsollaTokenRequest struct {
	Sandbox          bool              `json:"sandbox"`
	User             XsollaUser        `json:"user"`
	Purchase         XsollaPurchase    `json:"purchase"`
	Settings         XsollaSettings    `json:"settings"`
	CustomParameters map[string]string `json:"custom_parameters,omitempty"`
}

type XsollaUser struct {
	ID      XsollaValue `json:"id"`
	Country XsollaValue `json:"country"`
}

type XsollaValue struct {
	Value string `json:"value"`
}

type XsollaPurchase struct {
	Items []XsollaItem `json:"items"`
}

type XsollaItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type XsollaSettings struct {
	Currency string `json:"currency"`
}

type XsollaTokenResponse struct {
	Token string `json:"token"`
}

func (h *Handler) CreateXsollaPayment(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	if userID == "" {
		apierrors.Write(
			w,
			http.StatusUnauthorized,
			apierrors.CodeUnauthorized,
			"user ID is required",
		)
		return
	}

	var req XsollaPaymentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid body",
		)
		return
	}

	if req.Country == "" {
		req.Country = "US"
	}

	projectID := os.Getenv("XSOLLA_PROJECT_ID")
	merchantID := os.Getenv("XSOLLA_MERCHANT_ID")
	apiKey := os.Getenv("XSOLLA_API_KEY")

	if projectID == "" || merchantID == "" || apiKey == "" {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"Xsolla configuration is missing",
		)
		return
	}

	payload := XsollaTokenRequest{
		Sandbox: true,

		User: XsollaUser{
			ID: XsollaValue{
				Value: userID,
			},
			Country: XsollaValue{
				Value: req.Country,
			},
		},

		Purchase: XsollaPurchase{
			Items: []XsollaItem{
				{
					SKU:      xsollaNotebookSKU,
					Quantity: 1,
				},
			},
		},

		Settings: XsollaSettings{
			Currency: "USD",
		},

		CustomParameters: map[string]string{
			"user_id": userID,
			"sku":     xsollaNotebookSKU,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"failed to create Xsolla request",
		)
		return
	}

	url := fmt.Sprintf(
		"https://store.xsolla.com/api/v3/project/%s/admin/payment/token",
		projectID,
	)

	auth := base64.StdEncoding.EncodeToString(
		[]byte(projectID + ":" + apiKey),
	)

	xsollaReq, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		url,
		bytes.NewBuffer(body),
	)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"failed to create Xsolla request",
		)
		return
	}

	xsollaReq.Header.Set("Content-Type", "application/json")
	xsollaReq.Header.Set("Authorization", "Basic "+auth)

	client := &http.Client{}

	resp, err := client.Do(xsollaReq)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadGateway,
			apierrors.CodeInternal,
			"failed to contact Xsolla",
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// apierrors.Write(
		// 	w,
		// 	http.StatusBadGateway,
		// 	apierrors.CodeInternal,
		// 	"Xsolla payment token request failed",
		// )
		// return
		var xsollaError any

		if err := json.NewDecoder(resp.Body).Decode(&xsollaError); err != nil {
			xsollaError = "unable to decode Xsolla error response"
		}

		writeJSON(w, http.StatusBadGateway, map[string]any{
			"xsolla_status": resp.StatusCode,
			"xsolla_error":  xsollaError,
		})
		return
	}

	var tokenResponse XsollaTokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		apierrors.Write(
			w,
			http.StatusBadGateway,
			apierrors.CodeInternal,
			"invalid Xsolla response",
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"token": tokenResponse.Token,
		"url":   "https://sandbox-secure.xsolla.com/paystation4/?token=" + tokenResponse.Token,
	})
}
