package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// Dummy handler for testing
func dummyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Create router with middleware
func createRouter() *mux.Router {
	router := mux.NewRouter()
	router.Use(BlockHeightMiddleware)
	router.HandleFunc("/", dummyHandler).Methods("GET")
	return router
}

// Tests for BlockHeightMiddleware
func TestBlockHeightMiddleware(t *testing.T) {
	router := createRouter()

	tests := []struct {
		name           string
		queryParam     string
		expectedStatus int
		expectedHeader string
	}{
		{
			name:           "Valid height > 0",
			queryParam:     "123",
			expectedStatus: http.StatusOK,
			expectedHeader: "123",
		},
		{
			name:           "Height = 0",
			queryParam:     "0",
			expectedStatus: http.StatusOK,
			expectedHeader: "",
		},
		{
			name:           "Negative height",
			queryParam:     "-1",
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
		},
		{
			name:           "Non-numeric height",
			queryParam:     "abc",
			expectedStatus: http.StatusBadRequest,
			expectedHeader: "",
		},
		{
			name:           "No height parameter",
			queryParam:     "",
			expectedStatus: http.StatusOK,
			expectedHeader: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.queryParam != "" {
				q := req.URL.Query()
				q.Add("height", tc.queryParam)
				req.URL.RawQuery = q.Encode()
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)

			if tc.expectedHeader != "" {
				assert.Equal(t, tc.expectedHeader, req.Header.Get(grpctypes.GRPCBlockHeightHeader))
			} else {
				assert.Empty(t, rec.Header().Get(grpctypes.GRPCBlockHeightHeader))
			}
		})
	}
}
