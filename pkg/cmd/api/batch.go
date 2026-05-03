package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type batchRequest struct {
	Method  string                 `json:"method"`
	URL     string                 `json:"url"`
	Headers map[string]string      `json:"headers"`
	Body    map[string]interface{} `json:"body"`
}

// parseBatchInput reads API requests from an io.Reader, one per line.
// Each line is either a plain endpoint path (treated as GET) or a JSON object
// with method, url, headers, and body fields.
func parseBatchInput(r io.Reader) ([]batchRequest, error) {
	var requests []batchRequest
	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req batchRequest
		if line[0] == '{' {
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				return nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
			}
			if req.URL == "" {
				return nil, fmt.Errorf("line %d: missing \"url\" field", lineNum)
			}
		} else {
			req = batchRequest{URL: line}
		}

		if req.Method == "" {
			req.Method = "GET"
		}
		requests = append(requests, req)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

type batchResponseEnvelope struct {
	Status int             `json:"status"`
	Path   string          `json:"path"`
	Body   json.RawMessage `json:"body"`
}

func writeBatchResponse(w io.Writer, status int, path string, body []byte) error {
	env := batchResponseEnvelope{
		Status: status,
		Path:   path,
		Body:   json.RawMessage(body),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}
