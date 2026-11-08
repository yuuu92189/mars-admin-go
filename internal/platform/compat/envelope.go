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

// Envelope adapts canonical `/api/v1` responses to the legacy `{code,message,data}`
// contract at the HTTP boundary. Streaming and non-JSON responses are untouched.
func Envelope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLegacyAPI(r.URL.Path) || isWebSocket(r) {
			next.ServeHTTP(w, r)
			return
		}
		writer := newEnvelopeResponseWriter(w)
		next.ServeHTTP(writer, r)
		if writer.passthrough {
			return
		}
		contentType := writer.header.Get("Content-Type")
		if writer.status == http.StatusNoContent {
			copyHeader(w.Header(), writer.header)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(envelope{Code: http.StatusOK, Message: "ok", Msg: "ok", Data: nil})
			return
		}
		if !strings.Contains(strings.ToLower(contentType), "json") {
			writer.commitPassthrough()
			return
		}
		message := http.StatusText(writer.status)
		if writer.status >= 400 {
			var problem struct {
				Detail string `json:"detail"`
				Code   string `json:"code"`
			}
			if json.Unmarshal(writer.body.Bytes(), &problem) == nil && problem.Detail != "" {
				message = problem.Detail
			}
		}
		var data any
		if writer.status < 400 && writer.body.Len() > 0 {
			if err := json.Unmarshal(writer.body.Bytes(), &data); err != nil {
				data = writer.body.String()
			}
		}
		code := writer.status
		if code < 200 || code > 299 {
			code = writer.status
		} else {
			code = 200
			message = "ok"
		}
		copyHeader(w.Header(), writer.header)
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Legacy-Status", strconv.Itoa(writer.status))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(envelope{Code: code, Message: message, Msg: message, Data: data})
	})
}

type envelopeResponseWriter struct {
	destination http.ResponseWriter
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
	passthrough bool
}

func newEnvelopeResponseWriter(destination http.ResponseWriter) *envelopeResponseWriter {
	return &envelopeResponseWriter{
		destination: destination,
		header:      make(http.Header),
		status:      http.StatusOK,
	}
}

func (w *envelopeResponseWriter) Header() http.Header {
	if w.passthrough {
		return w.destination.Header()
	}
	return w.header
}

func (w *envelopeResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	if w.isNonJSON(w.header.Get("Content-Type")) {
		w.commitPassthrough()
	}
}

func (w *envelopeResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.status = http.StatusOK
	}
	if !w.passthrough && !isJSONResponse(w.header.Get("Content-Type"), data) {
		w.commitPassthrough()
	}
	if w.passthrough {
		return w.destination.Write(data)
	}
	return w.body.Write(data)
}

func (w *envelopeResponseWriter) isNonJSON(contentType string) bool {
	return contentType != "" && !strings.Contains(strings.ToLower(contentType), "json")
}

func isJSONResponse(contentType string, data []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "json") {
		return true
	}
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}

func (w *envelopeResponseWriter) commitPassthrough() {
	if w.passthrough {
		return
	}
	w.passthrough = true
	copyHeader(w.destination.Header(), w.header)
	w.destination.WriteHeader(w.status)
}

func (w *envelopeResponseWriter) Flush() {
	if !w.passthrough {
		if w.isNonJSON(w.header.Get("Content-Type")) {
			w.commitPassthrough()
		} else {
			return
		}
	}
	if flusher, ok := w.destination.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *envelopeResponseWriter) Unwrap() http.ResponseWriter {
	return w.destination
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
