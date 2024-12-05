package server

import (
	"net/http"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec/legacy"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
)

// BlockHeightMiddleware parses height query parameter and sets GRPCBlockHeightHeader
func BlockHeightMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		heightStr := r.FormValue("height")
		if heightStr != "" {
			height, err := strconv.ParseInt(heightStr, 10, 64)
			if err != nil {
				writeErrorResponse(w, http.StatusBadRequest, "syntax error")
				return
			}

			if height < 0 {
				writeErrorResponse(w, http.StatusBadRequest, "height must be equal or greater than zero")
				return
			}

			if height > 0 {
				r.Header.Set(grpctypes.GRPCBlockHeightHeader, heightStr)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// writeErrorResponse prepares and writes an HTTP error
func writeErrorResponse(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(legacy.Cdc.MustMarshalJSON(newErrorResponse(0, err)))
}

type errorResponse struct {
	Code  int    `json:"code,omitempty"`
	Error string `json:"error"`
}

func newErrorResponse(code int, err string) errorResponse {
	return errorResponse{Code: code, Error: err}
}
