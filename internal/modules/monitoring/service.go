package monitoring

import (
	"context"
	"math"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	pool    *pgxpool.Pool
	redis   *redis.Client
	started time.Time
}

type Snapshot struct {
	GoVersion      string         `json:"goVersion"`
	Goroutines     int            `json:"goroutines"`
	HeapAllocBytes uint64         `json:"heapAllocBytes"`
	HeapObjects    uint64         `json:"heapObjects"`
	GC             GCSnapshot     `json:"gc"`
	UptimeSeconds  int64          `json:"uptimeSeconds"`
	Database       DatabaseStatus `json:"database"`
	Redis          RedisStatus    `json:"redis"`
}

type GCSnapshot struct {
	LastGCUnixNano int64  `json:"lastGcUnixNano"`
	NumGC          uint32 `json:"numGc"`
	PauseTotalNs   uint64 `json:"pauseTotalNs"`
}

type DatabaseStatus struct {
	Reachable     bool  `json:"reachable"`
	MaxConns      int32 `json:"maxConns"`
	TotalConns    int32 `json:"totalConns"`
	IdleConns     int32 `json:"idleConns"`
	AcquiredConns int32 `json:"acquiredConns"`
}

type RedisStatus struct {
	Reachable bool `json:"reachable"`
}

type Dashboard struct {
	Users            int64 `json:"users"`
	UserCount        int64 `json:"userCount"`
	RoleCount        int64 `json:"roleCount"`
	MenuCount        int64 `json:"menuCount"`
	PermissionCount  int64 `json:"permissionCount"`
	OnlineSessions   int64 `json:"onlineSessions"`
	Files            int64 `json:"files"`
	Messages         int64 `json:"messages"`
	PublishedNotices int64 `json:"publishedNotices"`
	OutboxPending    int64 `json:"outboxPending"`
	OutboxDead       int64 `json:"outboxDead"`
}

func New(pool *pgxpool.Pool, redisClient *redis.Client, registry prometheus.Registerer) *Service {
	service := &Service{pool: pool, redis: redisClient, started: time.Now().UTC()}
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "mars_admin", Subsystem: "runtime", Name: "goroutines",
		Help: "Current number of goroutines.",
	}, func() float64 { return float64(runtime.NumGoroutine()) }))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "mars_admin", Subsystem: "runtime", Name: "heap_alloc_bytes",
		Help: "Current heap allocation in bytes.",
	}, func() float64 {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		return float64(stats.HeapAlloc)
	}))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "mars_admin", Subsystem: "database", Name: "max_connections",
		Help: "Maximum database connections in the pool.",
	}, func() float64 { return float64(pool.Stat().MaxConns()) }))
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "mars_admin", Subsystem: "database", Name: "total_connections",
		Help: "Total database connections in the pool.",
	}, func() float64 { return float64(pool.Stat().TotalConns()) }))
	return service
}

func (s *Service) Snapshot(ctx context.Context) Snapshot {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	dbStat := s.pool.Stat()
	lastGC := stats.LastGC
	if lastGC > math.MaxInt64 {
		lastGC = 0
	}
	snapshot := Snapshot{
		GoVersion: runtime.Version(), Goroutines: runtime.NumGoroutine(),
		HeapAllocBytes: stats.HeapAlloc, HeapObjects: stats.HeapObjects,
		GC:            GCSnapshot{LastGCUnixNano: int64(lastGC), NumGC: stats.NumGC, PauseTotalNs: stats.PauseTotalNs}, // #nosec G115 -- value is range checked immediately above.
		UptimeSeconds: int64(time.Since(s.started).Seconds()),
		Database: DatabaseStatus{
			MaxConns: dbStat.MaxConns(), TotalConns: dbStat.TotalConns(),
			IdleConns: dbStat.IdleConns(), AcquiredConns: dbStat.AcquiredConns(),
		},
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	snapshot.Database.Reachable = s.pool.Ping(checkCtx) == nil
	if s.redis != nil {
		snapshot.Redis.Reachable = s.redis.Ping(checkCtx).Err() == nil
	}
	return snapshot
}

func (s *Service) ForceGC() {
	debug.FreeOSMemory()
}

func (s *Service) Dashboard(ctx context.Context) (Dashboard, error) {
	var dashboard Dashboard
	queries := []struct {
		sql string
		dst *int64
	}{
		{`SELECT count(*) FROM identity_users WHERE deleted_at IS NULL`, &dashboard.Users},
		{`SELECT count(*) FROM identity_sessions WHERE revoked_at IS NULL AND expires_at > now()`, &dashboard.OnlineSessions},
		{`SELECT count(*) FROM storage_files WHERE deleted_at IS NULL`, &dashboard.Files},
		{`SELECT count(*) FROM messaging_messages`, &dashboard.Messages},
		{`SELECT count(*) FROM notification_notices WHERE status = 'published'`, &dashboard.PublishedNotices},
		{`SELECT count(*) FROM access_roles WHERE deleted_at IS NULL`, &dashboard.RoleCount},
		{`SELECT count(*) FROM access_menus`, &dashboard.MenuCount},
		{`SELECT count(*) FROM access_permissions`, &dashboard.PermissionCount},
		{`SELECT count(*) FROM outbox_events WHERE status IN ('pending', 'processing', 'failed')`, &dashboard.OutboxPending},
		{`SELECT count(*) FROM outbox_events WHERE status = 'dead'`, &dashboard.OutboxDead},
	}
	for _, query := range queries {
		if err := s.pool.QueryRow(ctx, query.sql).Scan(query.dst); err != nil {
			return Dashboard{}, err
		}
	}
	dashboard.UserCount = dashboard.Users
	return dashboard, nil
}
