package handlers

import (
	"checkout-api/apierrors"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32) // 32 byte for 256 bits, means 2^256 possible

	_, err := rand.Read(bytes) // CSPRNG
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil // convert to hex
}

// IssueJWT   	godoc
// @Summary      Logins the user
// @Description  Logins the user and gives tokens if correncly loginned
// @Tags         Auth
// @Produce      json
// @Param		 reqBody	body	RefreshRequest	true	"The refresh token details"
// @Success      200  {object}  AuthResponse
// @Failure      400  {object}  apierrors.ErrorDetail
// @Router       /token [get]
func (h *Handler) IssueJWT(w http.ResponseWriter, r *http.Request) {
	// TODO: implement issueing of new JWT with refresh token
	// check if refresh_token exists in the db and still active
	// generate a new JWT
	// generate a new random string (bonus: if you use a CSPRNG to generate a random sequence of bytes) as refresh_token
	// save new refresh token in db
	// deactivate old refresh token

	var req RefreshRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid request body",
		)
		return
	}

	userID, active, err := h.store.GetRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusUnauthorized,
			apierrors.CodeUnauthorized,
			"refresh token not found",
		)
		return
	}

	if !active {
		apierrors.Write(
			w,
			http.StatusUnauthorized,
			apierrors.CodeUnauthorized,
			"refresh token inactive",
		)
		return
	}

	jwtToken, err := GenerateJWT(userID)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"failed to generate JWT",
		)
		return
	}

	newRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"failed to generate refresh token",
		)
		return
	}

	err = h.store.SaveRefreshToken(r.Context(), userID, newRefreshToken)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"failed to save refresh token",
		)
		return
	}

	err = h.store.DeactivateRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			"failed to deactivate refresh token",
		)
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		JWT:          jwtToken,
		RefreshToken: newRefreshToken,
	})

}
