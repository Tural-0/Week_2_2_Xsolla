package handlers

import (
	"checkout-api/apierrors"
	"checkout-api/validation"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginUser   	godoc
// @Summary      Logins the user
// @Description  Logins the user and gives tokens if correncly loginned
// @Tags         Auth
// @Produce      json
// @Param		 reqBody	body	AuthRequest	true	"The login details"
// @Success      200  {object}  AuthResponse
// @Failure      400  {object}  apierrors.ErrorDetail
// @Router       /login [post]
func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid body",
		)
		return
	}

	req.Email = validation.Email(req.Email)

	if err := validation.RequiredString(req.Email, "Email"); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeValidationError,
			err.Error(),
		)
		return
	}

	if err := validation.RequiredString(req.Password, "Password"); err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeValidationError,
			err.Error(),
		)
		return
	}

	user, err := h.store.FindUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.Write(
				w,
				http.StatusNotFound,
				apierrors.CodeNotFound,
				"Invalid Credentials",
			)
			return
		}
		fmt.Printf("cannot query %q", err.Error())
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	err = bcrypt.CompareHashAndPassword(user.Hash, []byte(req.Password))
	if err != nil {
		apierrors.Write(
			w,
			http.StatusUnauthorized,
			apierrors.CodeUnauthorized,
			"Unauthorized",
		)
		return
	}
	// issue jwt
	//fifteenAfter := time.Now().Add(15 * time.Minute)
	//token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
	//	ExpiresAt: jwt.NewNumericDate(fifteenAfter),
	//	Subject:   strconv.Itoa(user.ID),
	//	IssuedAt:  jwt.NewNumericDate(time.Now()),
	//})

	//signedString, err := token.SignedString([]byte(SigningSecret))
	signedString, err := GenerateJWT(user.ID)
	if err != nil {
		fmt.Printf("cannot generate signed string %q", err.Error())
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	// store session(refresh token)
	// TODO: do it yourself
	// generate a random string(bonus: if you use a CSPRNG to generate a random sequence of bytes)
	// insert into refresh_tokens (token_value, is_active) values ("sOmERANdomlYGeNERATEDstRing", 1)
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	err = h.store.SaveRefreshToken(r.Context(), user.ID, refreshToken)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		JWT:          signedString,
		RefreshToken: refreshToken,
	})
}

// CreateUser   godoc
// @Summary      Creates a user
// @Description  Creates(signup) a user for later login use
// @Tags         Auth
// @Produce      json
// @Param		 reqBody	body	AuthRequest	true	"The login details"
// @Success      201
// @Failure      400  {object}  apierrors.ErrorDetail
// @Router       /signup [post]
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusBadRequest,
			apierrors.CodeInvalidRequest,
			"invalid body",
		)
		return
	}

	// validate email
	req.Email = validation.Email(req.Email)

	if err := validation.RequiredString(req.Email, "Email"); err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeInvalidRequest,
			err.Error(),
		)
		return
	}

	if err := validation.EmailCheck(req.Email); err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeInvalidRequest,
			err.Error(),
		)
		return
	}

	if err := validation.PasswordCheck(req.Password); err != nil {
		apierrors.Write(
			w,
			http.StatusUnprocessableEntity,
			apierrors.CodeInvalidRequest,
			err.Error(),
		)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	err = h.store.SaveUser(r.Context(), req.Email, hash)
	if err != nil {
		apierrors.Write(
			w,
			http.StatusInternalServerError,
			apierrors.CodeInternal,
			err.Error(),
		)
		return
	}

	writeJSON(w, http.StatusCreated, nil)
}
