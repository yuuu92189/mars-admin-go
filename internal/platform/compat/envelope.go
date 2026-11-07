package compat

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Msg     string `json:"msg,omitempty"`
	Data    any    `json:"data"`
}

type bufferedResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{header: make(http.Header), status: http.StatusOK}
}

func (b *bufferedResponse) Header() http.Header            { return b.header }
func (b *bufferedResponse) WriteHeader(status int)         { b.status = status }
func (b *bufferedResponse) Write(data []byte) (int, error) { return b.body.Write(data) }

// Envelope adapts canonical `/api/v1` responses to the legacy `{code,message,data}`
// contract at the HTTP boundary. Streaming and non-JSON responses are untouched.
func Envelope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLegacyAPI(r.URL.Path) || isWebSocket(r) {
			next.ServeHTTP(w, r)
			return
		}
		buffer := newBufferedResponse()
		next.ServeHTTP(buffer, r)
		contentType := buffer.header.Get("Content-Type")
		if buffer.status == http.StatusNoContent {
			copyHeader(w.Header(), buffer.header)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(envelope{Code: http.StatusOK, Message: "ok", Msg: "ok", Data: nil})
			return
		}
		if !strings.Contains(contentType, "json") {
			copyHeader(w.Header(), buffer.header)
			w.WriteHeader(buffer.status)
			_, _ = w.Write(buffer.body.Bytes())
			return
		}
		message := http.StatusText(buffer.status)
		if buffer.status >= 400 {
			var problem struct {
				Detail string `json:"detail"`
				Code   string `json:"code"`
			}
			if json.Unmarshal(buffer.body.Bytes(), &problem) == nil && problem.Detail != "" {
				message = problem.Detail
			}
		}
		var data any
		if buffer.status < 400 && buffer.body.Len() > 0 {
			if err := json.Unmarshal(buffer.body.Bytes(), &data); err != nil {
				data = buffer.body.String()
			}
		}
		code := buffer.status
		if code < 200 || code > 299 {
			code = buffer.status
		} else {
			code = 200
			message = "ok"
		}
		copyHeader(w.Header(), buffer.header)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Legacy-Status", strconv.Itoa(buffer.status))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(envelope{Code: code, Message: message, Msg: message, Data: data})
	})
}

func isLegacyAPI(path string) bool {
	if !strings.HasPrefix(path, "/api/") {
		return false
	}
	return !strings.HasPrefix(path, "/api/v1/")
}

func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
