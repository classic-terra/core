package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// TxLogsMiddleware intercepts tx query responses and reconstructs the deprecated
// `logs` field from the `events` field for backwards compatibility.
// This is needed because Cosmos SDK 0.50+ no longer populates the logs field.
func TxLogsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only process GET requests to tx endpoints
		if r.Method != http.MethodGet || !isTxQueryEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Capture the response
		recorder := &responseRecorder{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
			headers:        make(http.Header),
		}

		next.ServeHTTP(recorder, r)

		// Only transform successful responses
		if recorder.statusCode != http.StatusOK {
			// Forward original headers
			for k, v := range recorder.headers {
				w.Header()[k] = v
			}
			w.WriteHeader(recorder.statusCode)
			w.Write(recorder.body.Bytes())
			return
		}

		// Try to transform the response
		transformed, err := transformTxResponse(recorder.body.Bytes())
		if err != nil {
			// If transformation fails, return original response
			for k, v := range recorder.headers {
				w.Header()[k] = v
			}
			w.WriteHeader(recorder.statusCode)
			w.Write(recorder.body.Bytes())
			return
		}

		// Write transformed response
		for k, v := range recorder.headers {
			if k != "Content-Length" {
				w.Header()[k] = v
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(transformed)))
		w.WriteHeader(recorder.statusCode)
		w.Write(transformed)
	})
}

// isTxQueryEndpoint checks if the path is a tx query endpoint
func isTxQueryEndpoint(path string) bool {
	// Match /cosmos/tx/v1beta1/txs and /cosmos/tx/v1beta1/txs/{hash}
	return strings.HasPrefix(path, "/cosmos/tx/v1beta1/txs")
}

// responseRecorder captures the response for modification
type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	headers    http.Header
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

// transformTxResponse transforms the tx response to include logs
func transformTxResponse(body []byte) ([]byte, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	// Handle single tx response (GetTx)
	if txResponse, ok := response["tx_response"].(map[string]interface{}); ok {
		reconstructLogs(txResponse)
	}

	// Handle multiple tx responses (GetTxsEvent)
	if txResponses, ok := response["tx_responses"].([]interface{}); ok {
		for _, txResp := range txResponses {
			if txRespMap, ok := txResp.(map[string]interface{}); ok {
				reconstructLogs(txRespMap)
			}
		}
	}

	return json.Marshal(response)
}

// reconstructLogs rebuilds the logs field from events
func reconstructLogs(txResponse map[string]interface{}) {
	// Check if logs is already populated
	if logs, ok := txResponse["logs"].([]interface{}); ok && len(logs) > 0 {
		return
	}

	// Get events from the response
	events, ok := txResponse["events"].([]interface{})
	if !ok || len(events) == 0 {
		return
	}

	// Group events by msg_index
	eventsByMsgIndex := make(map[int][]map[string]interface{})
	maxMsgIndex := 0

	for _, event := range events {
		eventMap, ok := event.(map[string]interface{})
		if !ok {
			continue
		}

		msgIndex := extractMsgIndex(eventMap)
		if msgIndex > maxMsgIndex {
			maxMsgIndex = msgIndex
		}

		// Create a copy of the event without msg_index in attributes for logs format
		eventCopy := copyEventForLogs(eventMap)
		eventsByMsgIndex[msgIndex] = append(eventsByMsgIndex[msgIndex], eventCopy)
	}

	// Build logs array
	logs := make([]map[string]interface{}, 0, maxMsgIndex+1)
	for i := 0; i <= maxMsgIndex; i++ {
		msgEvents := eventsByMsgIndex[i]
		if len(msgEvents) == 0 {
			continue
		}

		log := map[string]interface{}{
			"msg_index": i,
			"log":       "",
			"events":    msgEvents,
		}
		logs = append(logs, log)
	}

	if len(logs) > 0 {
		txResponse["logs"] = logs
	}
}

// extractMsgIndex extracts the msg_index from event attributes
func extractMsgIndex(event map[string]interface{}) int {
	attributes, ok := event["attributes"].([]interface{})
	if !ok {
		return 0
	}

	for _, attr := range attributes {
		attrMap, ok := attr.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for msg_index in the attribute itself (SDK 0.50+ format)
		if msgIndex, ok := attrMap["msg_index"]; ok {
			return toInt(msgIndex)
		}

		// Also check for msg_index as a key-value pair
		if key, ok := attrMap["key"].(string); ok && key == "msg_index" {
			if value, ok := attrMap["value"].(string); ok {
				if idx, err := strconv.Atoi(value); err == nil {
					return idx
				}
			}
		}
	}

	return 0
}

// copyEventForLogs creates a copy of the event suitable for the logs format
func copyEventForLogs(event map[string]interface{}) map[string]interface{} {
	eventCopy := make(map[string]interface{})

	if eventType, ok := event["type"].(string); ok {
		eventCopy["type"] = eventType
	}

	if attributes, ok := event["attributes"].([]interface{}); ok {
		attrsCopy := make([]map[string]interface{}, 0, len(attributes))
		for _, attr := range attributes {
			if attrMap, ok := attr.(map[string]interface{}); ok {
				attrCopy := make(map[string]interface{})
				if key, ok := attrMap["key"].(string); ok {
					attrCopy["key"] = key
				}
				if value, ok := attrMap["value"]; ok {
					attrCopy["value"] = value
				}
				attrsCopy = append(attrsCopy, attrCopy)
			}
		}
		eventCopy["attributes"] = attrsCopy
	}

	return eventCopy
}

// toInt converts an interface to int
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}
