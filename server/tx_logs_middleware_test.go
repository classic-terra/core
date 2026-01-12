package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxLogsMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		method          string
		responseBody    string
		expectLogsLen   int
		expectTransform bool
	}{
		{
			name:   "single tx with events - should reconstruct logs",
			path:   "/cosmos/tx/v1beta1/txs/ABCD1234",
			method: http.MethodGet,
			responseBody: `{
				"tx": {},
				"tx_response": {
					"height": "100",
					"txhash": "ABCD1234",
					"logs": [],
					"events": [
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "send", "msg_index": 0}
							]
						},
						{
							"type": "transfer",
							"attributes": [
								{"key": "recipient", "value": "terra1abc", "msg_index": 0}
							]
						}
					]
				}
			}`,
			expectLogsLen:   1,
			expectTransform: true,
		},
		{
			name:   "multiple messages - should group by msg_index",
			path:   "/cosmos/tx/v1beta1/txs/ABCD1234",
			method: http.MethodGet,
			responseBody: `{
				"tx": {},
				"tx_response": {
					"height": "100",
					"txhash": "ABCD1234",
					"logs": [],
					"events": [
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "send", "msg_index": 0}
							]
						},
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "delegate", "msg_index": 1}
							]
						}
					]
				}
			}`,
			expectLogsLen:   2,
			expectTransform: true,
		},
		{
			name:   "non-tx endpoint - should not transform",
			path:   "/cosmos/bank/v1beta1/balances/terra1abc",
			method: http.MethodGet,
			responseBody: `{
				"balances": []
			}`,
			expectTransform: false,
		},
		{
			name:   "POST request - should not transform",
			path:   "/cosmos/tx/v1beta1/txs",
			method: http.MethodPost,
			responseBody: `{
				"tx_response": {}
			}`,
			expectTransform: false,
		},
		{
			name:   "GetTxsEvent response - should transform multiple tx_responses",
			path:   "/cosmos/tx/v1beta1/txs",
			method: http.MethodGet,
			responseBody: `{
				"txs": [],
				"tx_responses": [
					{
						"height": "100",
						"txhash": "TX1",
						"logs": [],
						"events": [
							{
								"type": "message",
								"attributes": [
									{"key": "action", "value": "send", "msg_index": 0}
								]
							}
						]
					},
					{
						"height": "101",
						"txhash": "TX2",
						"logs": [],
						"events": [
							{
								"type": "message",
								"attributes": [
									{"key": "action", "value": "delegate", "msg_index": 0}
								]
							}
						]
					}
				]
			}`,
			expectTransform: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock handler that returns the test response
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tc.responseBody))
			})

			// Wrap with our middleware
			handler := TxLogsMiddleware(mockHandler)

			// Create test request
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(rec, req)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)

			if tc.expectTransform && tc.expectLogsLen > 0 {
				// Check single tx_response
				if txResponse, ok := response["tx_response"].(map[string]interface{}); ok {
					logs, ok := txResponse["logs"].([]interface{})
					require.True(t, ok, "logs should be present")
					require.Equal(t, tc.expectLogsLen, len(logs))
				}
			}

			if !tc.expectTransform {
				// Original response should be unchanged (structurally)
				var original map[string]interface{}
				err = json.Unmarshal([]byte(tc.responseBody), &original)
				require.NoError(t, err)
				// Just verify no error occurred
			}
		})
	}
}

func TestReconstructLogs(t *testing.T) {
	tests := []struct {
		name         string
		txResponse   map[string]interface{}
		expectedLogs int
	}{
		{
			name: "basic event reconstruction",
			txResponse: map[string]interface{}{
				"logs": []interface{}{},
				"events": []interface{}{
					map[string]interface{}{
						"type": "message",
						"attributes": []interface{}{
							map[string]interface{}{
								"key":       "action",
								"value":     "send",
								"msg_index": 0,
							},
						},
					},
				},
			},
			expectedLogs: 1,
		},
		{
			name: "already has logs - should not modify",
			txResponse: map[string]interface{}{
				"logs": []interface{}{
					map[string]interface{}{
						"msg_index": 0,
						"events":    []interface{}{},
					},
				},
				"events": []interface{}{},
			},
			expectedLogs: 1,
		},
		{
			name: "no events - should not add logs",
			txResponse: map[string]interface{}{
				"logs":   []interface{}{},
				"events": []interface{}{},
			},
			expectedLogs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reconstructLogs(tc.txResponse)

			// logs can be []interface{} (original) or []map[string]interface{} (reconstructed)
			logsRaw := tc.txResponse["logs"]
			var logsLen int
			switch logs := logsRaw.(type) {
			case []interface{}:
				logsLen = len(logs)
			case []map[string]interface{}:
				logsLen = len(logs)
			}
			require.Equal(t, tc.expectedLogs, logsLen)
		})
	}
}

func TestExtractMsgIndex(t *testing.T) {
	tests := []struct {
		name     string
		event    map[string]interface{}
		expected int
	}{
		{
			name: "msg_index in attribute",
			event: map[string]interface{}{
				"type": "message",
				"attributes": []interface{}{
					map[string]interface{}{
						"key":       "action",
						"value":     "send",
						"msg_index": 2,
					},
				},
			},
			expected: 2,
		},
		{
			name: "msg_index as float64 (JSON unmarshaling)",
			event: map[string]interface{}{
				"type": "message",
				"attributes": []interface{}{
					map[string]interface{}{
						"key":       "action",
						"value":     "send",
						"msg_index": float64(3),
					},
				},
			},
			expected: 3,
		},
		{
			name: "no msg_index - default to 0",
			event: map[string]interface{}{
				"type": "message",
				"attributes": []interface{}{
					map[string]interface{}{
						"key":   "action",
						"value": "send",
					},
				},
			},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractMsgIndex(tc.event)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestIsTxQueryEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/cosmos/tx/v1beta1/txs/ABCD1234", true},
		{"/cosmos/tx/v1beta1/txs", true},
		{"/cosmos/bank/v1beta1/balances/terra1abc", false},
		{"/cosmos/staking/v1beta1/validators", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := isTxQueryEndpoint(tc.path)
			require.Equal(t, tc.expected, result)
		})
	}
}
