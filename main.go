package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"checkout-api/handlers"
	"checkout-api/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	connString := os.Getenv("CON_STRING")

	ctx := context.Background()

	////////////////////////////////////////////////////

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	fmt.Println("successfully connected to db")

	////////////////////////////////////////////////////

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-User-ID", "Idempotency-Key"},
	})

	// s := store.NewInMemStore()
	postgresStore := store.NewPostgresStore(pool)
	h := handlers.NewHandler(postgresStore)
	mux := http.NewServeMux()

	// cart
	mux.HandleFunc("POST /user/carts", h.CreateUserCart)
	//mux.Handle(
	//	"POST /user/carts",
	//	middleware.JWTMiddleware(http.HandlerFunc(h.CreateUserCart)),
	//)

	mux.HandleFunc("GET /user/cart", h.GetUserCart)
	//mux.Handle(
	//	"GET /user/cart",
	//	middleware.JWTMiddleware(http.HandlerFunc(h.GetUserCart)),
	//)

	mux.HandleFunc("PATCH /user/cart/items/{item_id}", h.UpsertCartItem)
	//mux.Handle(
	//	"PATCH /user/cart/items/{item_id}",
	//	middleware.JWTMiddleware(http.HandlerFunc(h.UpsertCartItem)),
	//)

	mux.HandleFunc("DELETE /user/cart/items/{item_id}", h.RemoveCartItem)
	//mux.Handle(
	//	"DELETE /user/cart/items/{item_id}",
	//	middleware.JWTMiddleware(http.HandlerFunc(h.RemoveCartItem)),
	//)

	mux.HandleFunc("DELETE /user/cart", h.DeleteUserCart)

	// orders
	mux.HandleFunc("POST /orders", h.CreateOrder)
	//mux.Handle(
	//	"POST /orders",
	//	middleware.JWTMiddleware(http.HandlerFunc(h.CreateOrder)),
	//)
	mux.HandleFunc("GET /user/orders", h.GetUserOrders)

	// items
	mux.HandleFunc("GET /items", h.GetItems)
	mux.HandleFunc("GET /items/{item_id}", h.GetItemByID)
	mux.HandleFunc("GET /itemQuantity/{item_id}", h.GetItemQuantityById) // auth for this

	// users
	mux.HandleFunc("POST /signup", h.CreateUser)
	mux.HandleFunc("POST /login", h.LoginUser)
	mux.HandleFunc("GET /token", h.IssueJWT)

	fmt.Println("Server starting on :8080")
	handler := c.Handler(mux)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
