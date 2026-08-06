# Checkout API

Xsolla School training project — a simplified checkout API built with Go.

Students build this service incrementally across lectures, starting from a basic HTTP server and evolving it into a production-ready system with persistence, authentication, observability, and more.

## Prerequisites

- Go 1.21+
- curl or Postman (for testing endpoints)
- A code editor (VS Code, GoLand, etc.)

## Quick Start

```bash
go run main.go
```

The server starts on http://localhost:8080.

## API Endpoints

### GET /items

Returns all available items.

```bash
curl http://localhost:8080/items
```

### POST /orders

Creates an order with mock payment processing.

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "items": [{"item_id": 1, "quantity": 2}]}'
```

## Running Tests

```bash
go test -v ./handlers/
```

## Branch Guide

Each lecture has two branches:

| Branch | Purpose |
|--------|---------|
| `week-XX/lecture-XX` | **Starter** — scaffold with TODOs and pre-written tests. Fork from here at the start of class. |
| `week-XX/lecture-XX-final` | **Final** — completed code matching the lecture. Compare your work against this. |

### Available Branches

- `week-01/lecture-01` — Intro to HTTP and JSON APIs (starter)
- `week-01/lecture-01-final` — Intro to HTTP and JSON APIs (completed)

## Project Structure (Week 1)

```
checkout-api/
├── main.go                  # HTTP server entry point
├── models/models.go         # Domain models (Item, LineItem, Order)
├── store/store.go           # In-memory data storage
├── handlers/handlers.go     # HTTP route handlers
└── handlers/handlers_test.go # Table-driven handler tests
```

## License

Internal — Xsolla School use only.
