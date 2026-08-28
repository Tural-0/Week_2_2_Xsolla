package middleware

import (
	"checkout-api/apierrors"
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var ctx context.Context

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			apierrors.Write(
				w,
				http.StatusUnauthorized,
				apierrors.CodeUnauthorized,
				"missing Authorizartion header",
			)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			apierrors.Write(
				w,
				http.StatusUnauthorized,
				apierrors.CodeUnauthorized,
				"invalid Authorization header",
			)
			return
		}

		tokenString := parts[1]

		_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { // signature check
				return nil, jwt.ErrTokenSignatureInvalid
			}

			claims := token.Claims.(jwt.MapClaims) // userId, exp, iat

			ctx = context.WithValue(r.Context(), "userID", int(claims["user_id"].(float64)))
			// converts to int for later use

			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil {
			apierrors.Write(
				w,
				http.StatusUnauthorized,
				apierrors.CodeUnauthorized,
				"invalid token",
			)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
