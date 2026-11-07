package monitoring

import (
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"mars-admin-go/internal/platform/httpserver"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
	auth    func(http.Handler) http.Handler
	logger  *slog.Logger
	metrics *httpserver.HTTPMetrics
}

func NewHandler(service *Service, auth func(http.Handler) http.Handler, logger *slog.Logger, metrics ...*httpserver.HTTPMetrics) *Handler {
	handler := &Handler{service: service, auth: auth, logger: logger}
	if len(metrics) > 0 {
		handler.metrics = metrics[0]
	}
	return handler
}

func (h *Handler) Routes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.Get("/snapshot", h.snapshot)
	router.Get("/dashboard", h.dashboard)
	return router
}

func (h *Handler) LegacyRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth)
	router.Get("/stats", h.dashboard)
	router.Get("/snapshot", h.snapshot)
	router.Get("/info", h.legacyInfo)
	return router
}

func (h *Handler) LegacyCacheRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth, httpserver.RequirePermission("audit:read"))
	router.Get("/stats", h.cacheStats)
	router.Get("/info", h.cacheStats)
	router.Get("/keys", h.cacheKeys)
	router.Get("/value", h.cacheValue)
	router.Delete("/", h.cacheDelete)
	return router
}

func (h *Handler) LegacyAPIAccessRoutes() chi.Router {
	router := chi.NewRouter()
	router.Use(h.auth, httpserver.RequirePermission("audit:read"))
	router.Get("/page", h.apiAccessPage)
	router.Get("/statistics", h.apiAccessStatistics)
	return router
}

func (h *Handler) apiAccessPage(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpserver.ParsePagination(r, 100)
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	calls := h.filteredAPICalls(r)
	start := (page - 1) * pageSize
	if start > len(calls) {
		start = len(calls)
	}
	end := start + pageSize
	if end > len(calls) {
		end = len(calls)
	}
	items := make([]map[string]any, 0, end-start)
	for _, call := range calls[start:end] {
		items = append(items, apiCallObject(call))
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"list": items, "records": items, "items": items,
		"total": len(calls), "page": page, "pageSize": pageSize,
	})
}

func (h *Handler) apiAccessStatistics(w http.ResponseWriter, r *http.Request) {
	calls := h.filteredAPICalls(r)
	success, failure := 0, 0
	daily := make(map[string]map[string]int)
	paths := make(map[string]int)
	methods := make(map[string]int)
	for _, call := range calls {
		if call.StatusCode >= 400 {
			failure++
		} else {
			success++
		}
		date := call.StartedAt.Format("2006-01-02")
		if daily[date] == nil {
			daily[date] = map[string]int{"total": 0, "success": 0, "fail": 0}
		}
		daily[date]["total"]++
		if call.StatusCode >= 400 {
			daily[date]["fail"]++
		} else {
			daily[date]["success"]++
		}
		paths[call.Path]++
		methods[call.Method]++
	}
	type pathCount struct {
		Path  string `json:"apiPath"`
		Count int    `json:"count"`
	}
	top := make([]pathCount, 0, len(paths))
	for path, count := range paths {
		top = append(top, pathCount{Path: path, Count: count})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].Count > top[j].Count })
	if len(top) > 10 {
		top = top[:10]
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"totalCount": len(calls), "successCount": success, "failCount": failure,
		"dailyStats": daily, "topPaths": top, "methodCount": methods,
	})
}

func (h *Handler) filteredAPICalls(r *http.Request) []httpserver.APICall {
	if h.metrics == nil {
		return []httpserver.APICall{}
	}
	calls := h.metrics.RecentAPICalls()
	sort.Slice(calls, func(i, j int) bool { return calls[i].StartedAt.After(calls[j].StartedAt) })
	pathFilter := r.URL.Query().Get("apiPath")
	methodFilter := strings.ToUpper(r.URL.Query().Get("method"))
	successFilter := r.URL.Query().Get("success")
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")
	result := make([]httpserver.APICall, 0, len(calls))
	for _, call := range calls {
		if pathFilter != "" && !strings.Contains(call.Path, pathFilter) {
			continue
		}
		if methodFilter != "" && call.Method != methodFilter {
			continue
		}
		success := call.StatusCode < 400
		if successFilter != "" {
			expected := successFilter == "1" || strings.EqualFold(successFilter, "true")
			if success != expected {
				continue
			}
		}
		date := call.StartedAt.Format("2006-01-02")
		if startDate != "" && date < startDate {
			continue
		}
		if endDate != "" && date > endDate {
			continue
		}
		result = append(result, call)
	}
	return result
}

func apiCallObject(call httpserver.APICall) map[string]any {
	success := 0
	if call.StatusCode < 400 {
		success = 1
	}
	return map[string]any{
		"id": call.ID, "startTime": call.StartedAt.Format("2006-01-02 15:04:05"),
		"endTime": call.StartedAt.Add(time.Duration(call.DurationMS) * time.Millisecond).Format("2006-01-02 15:04:05"),
		"apiPath": call.Path, "method": call.Method, "statusCode": call.StatusCode,
		"success": success, "costTime": call.DurationMS, "ip": call.IP, "userId": call.UserID,
	}
}

func (h *Handler) cacheStats(w http.ResponseWriter, r *http.Request) {
	info, err := h.service.redis.Info(r.Context(), "memory", "stats", "clients").Result()
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	values := parseRedisInfo(info)
	usedMemory, _ := strconv.ParseInt(values["used_memory"], 10, 64)
	maxMemory, _ := strconv.ParseInt(values["maxmemory"], 10, 64)
	if maxMemory == 0 {
		maxMemory, _ = strconv.ParseInt(values["total_system_memory"], 10, 64)
	}
	ops, _ := strconv.ParseInt(values["instantaneous_ops_per_sec"], 10, 64)
	clients, _ := strconv.ParseInt(values["connected_clients"], 10, 64)
	hits, _ := strconv.ParseInt(values["keyspace_hits"], 10, 64)
	misses, _ := strconv.ParseInt(values["keyspace_misses"], 10, 64)
	hitRate := float64(0)
	if hits+misses > 0 {
		hitRate = float64(hits) / float64(hits+misses) * 100
	}
	httpserver.WriteJSON(w, http.StatusOK, map[string]any{
		"usedMemory": usedMemory, "maxMemory": maxMemory, "ops": ops,
		"hitRate": hitRate, "connectedClients": clients,
		"raw": values,
	})
}

func (h *Handler) cacheKeys(w http.ResponseWriter, r *http.Request) {
	pattern := r.URL.Query().Get("pattern")
	if pattern == "" {
		pattern = "*"
	}
	keys := make([]string, 0)
	iterator := h.service.redis.Scan(r.Context(), 0, pattern, 100).Iterator()
	for iterator.Next(r.Context()) && len(keys) < 500 {
		keys = append(keys, iterator.Val())
	}
	if err := iterator.Err(); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, keys)
}

func (h *Handler) cacheValue(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"detail": "Cache key is required."})
		return
	}
	value, err := h.service.redis.Get(r.Context(), key).Result()
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, value)
}

func (h *Handler) cacheDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		httpserver.WriteJSON(w, http.StatusBadRequest, map[string]string{"detail": "Cache key is required."})
		return
	}
	if err := h.service.redis.Del(r.Context(), key).Err(); err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseRedisInfo(value string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, item, ok := strings.Cut(line, ":")
		if ok {
			result[key] = item
		}
	}
	return result
}

func (h *Handler) legacyInfo(w http.ResponseWriter, r *http.Request) {
	snapshot := h.service.Snapshot(r.Context())
	hostname, _ := os.Hostname()
	started := time.Now().UTC().Add(-time.Duration(snapshot.UptimeSeconds) * time.Second)
	response := map[string]any{
		"cpu": map[string]any{
			"name": runtime.GOOS + " " + runtime.GOARCH, "arch": runtime.GOARCH,
			"availableProcessors": runtime.NumCPU(), "systemLoadAverage": 0,
		},
		"memory": map[string]any{
			"heapInit": "0 MB", "heapUsed": formatBytes(snapshot.HeapAllocBytes),
			"heapMax":     formatBytes(maxMemoryBytes(snapshot.HeapAllocBytes)),
			"nonHeapUsed": formatBytes(snapshot.HeapAllocBytes),
		},
		"jvm": map[string]any{
			"name": "Go runtime", "version": snapshot.GoVersion,
			"startTime": started.Format("2006-01-02 15:04:05"),
			"uptime":    formatDuration(snapshot.UptimeSeconds),
		},
		"sys": map[string]any{
			"hostName": hostname, "hostAddress": "127.0.0.1",
			"osName": runtime.GOOS, "osVersion": runtime.GOARCH,
		},
		"sysFiles": []map[string]any{},
		"snapshot": snapshot,
	}
	httpserver.WriteJSON(w, http.StatusOK, response)
}

func formatBytes(value uint64) string {
	const megabyte = 1024 * 1024
	return strconv.FormatFloat(float64(value)/float64(megabyte), 'f', 2, 64) + " MB"
}

func maxMemoryBytes(used uint64) uint64 {
	minimum := uint64(64 * 1024 * 1024)
	if used < minimum {
		return minimum
	}
	return used * 2
}

func formatDuration(seconds int64) string {
	duration := time.Duration(seconds) * time.Second
	days := int64(duration / (24 * time.Hour))
	duration -= time.Duration(days) * 24 * time.Hour
	hours := int64(duration / time.Hour)
	duration -= time.Duration(hours) * time.Hour
	minutes := int64(duration / time.Minute)
	if days > 0 {
		return strconv.FormatInt(days, 10) + "天" + strconv.FormatInt(hours, 10) + "小时"
	}
	if hours > 0 {
		return strconv.FormatInt(hours, 10) + "小时" + strconv.FormatInt(minutes, 10) + "分钟"
	}
	return strconv.FormatInt(minutes, 10) + "分钟"
}

func (h *Handler) snapshot(w http.ResponseWriter, r *http.Request) {
	httpserver.WriteJSON(w, http.StatusOK, h.service.Snapshot(r.Context()))
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Dashboard(r.Context())
	if err != nil {
		httpserver.WriteError(w, r, h.logger, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, data)
}
