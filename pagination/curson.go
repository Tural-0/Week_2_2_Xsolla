package pagination

import (
	"errors"
	"net/http"
	"strconv"
)

var ErrInvalidCursor = errors.New("cursor must be a positive integer")

type CursorParams struct {
	Limit  int
	Cursor *int
}

func ParseCursor(r *http.Request) (CursorParams, error) {
	params := CursorParams{
		Limit: DefaultLimit,
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return CursorParams{}, ErrInvalidLimit
		}

		params.Limit = limit
	}

	if params.Limit <= 0 {
		return CursorParams{}, ErrInvalidLimit
	}

	if params.Limit > MaxLimit {
		params.Limit = MaxLimit
	}

	cursorStr := r.URL.Query().Get("cursor")
	if cursorStr == "" {
		return params, nil
	}

	cursor, err := strconv.Atoi(cursorStr)
	if err != nil || cursor <= 0 {
		return CursorParams{}, ErrInvalidCursor
	}

	params.Cursor = &cursor

	return params, nil
}
