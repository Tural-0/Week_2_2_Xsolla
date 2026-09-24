package handlers

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

type XsollaWebhook struct {
	NotificationType string              `json:"notification_type"`
	User             XsollaWebhookUser   `json:"user"`
	Items            []XsollaWebhookItem `json:"items"`
	Billing          XsollaBilling       `json:"billing"`
}

type XsollaWebhookUser struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
}

type XsollaWebhookItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type XsollaBilling struct {
	Transaction XsollaTransaction `json:"transaction"`
}

type XsollaTransaction struct {
	ID string `json:"id"`
}

func (h *Handler) XsollaWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if !verifyXsollaWebhook(body, r.Header.Get("Authorization")) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var webhook XsollaWebhook

	if err := json.Unmarshal(body, &webhook); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	switch webhook.NotificationType {
	case "user_validation":
		w.WriteHeader(http.StatusOK)
		return

	case "order_paid":
		if err := h.processXsollaPayment(r, webhook); err != nil {
			//http.Error(w, "payment processing failed", http.StatusInternalServerError)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return

	default:
		w.WriteHeader(http.StatusOK)
		return
	}
}

func verifyXsollaWebhook(body []byte, authorization string) bool {
	const prefix = "Signature "

	if len(authorization) <= len(prefix) ||
		authorization[:len(prefix)] != prefix {
		return false
	}

	secretKey := os.Getenv("XSOLLA_WEBHOOK")

	hash := sha1.New()
	hash.Write(body)
	hash.Write([]byte(secretKey))

	expected := hex.EncodeToString(hash.Sum(nil))
	actual := authorization[len(prefix):]

	return subtle.ConstantTimeCompare(
		[]byte(actual),
		[]byte(expected),
	) == 1
}

func (h *Handler) processXsollaPayment(
	r *http.Request,
	webhook XsollaWebhook,
) error {
	transactionID := webhook.Billing.Transaction.ID

	if transactionID == "" {
		return fmt.Errorf("missing transaction ID")
	}

	if webhook.User.ExternalID == "" {
		return fmt.Errorf("missing user ID")
	}

	userID, err := strconv.Atoi(webhook.User.ExternalID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if len(webhook.Items) == 0 {
		return fmt.Errorf("no purchased items")
	}

	for _, item := range webhook.Items {
		if item.SKU == "" || item.Quantity <= 0 {
			return fmt.Errorf("invalid purchased item")
		}

		itemID, err := xsollaItemID(item.SKU)
		if err != nil {
			return err
		}

		_, err = h.store.ProcessXsollaPayment(
			r.Context(),
			transactionID,
			userID,
			item.SKU,
			itemID,
			item.Quantity,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func xsollaItemID(sku string) (int, error) {
	switch sku {
	case "NotebookA5":
		return 7, nil
	default:
		return 0, fmt.Errorf("unknown Xsolla SKU: %s", sku)
	}
}
