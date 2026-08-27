package pagination

import (
	"errors"
	"net/http"
	"strconv"
)

var (
	ErrInvalidLimit  = errors.New("limit must be a positive integer")
	ErrInvalidOffset = errors.New("offset must be zero or greater")
)

const (
	DefaultLimit = 10
	MaxLimit     = 100
)

type OffsetParams struct {
	Limit  int
	Offset int
}

func ParseOffset(r *http.Request) (OffsetParams, error) {
	params := OffsetParams{
		Limit:  DefaultLimit,
		Offset: 0,
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return OffsetParams{}, err
		}

		params.Limit = limit
	}

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return OffsetParams{}, err
		}

		params.Offset = offset
	}

	if params.Limit <= 0 {
		return OffsetParams{}, ErrInvalidLimit
	}

	if params.Limit > MaxLimit {
		params.Limit = MaxLimit
	}

	if params.Offset < 0 {
		return OffsetParams{}, ErrInvalidOffset
	}

	return params, nil
}
