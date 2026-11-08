package monitoring

import (
	"context"
	"math"
	"runtime"
	"runtime/debug"
	"strings"
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
	Users            int64               `json:"users"`
	UserCount        int64               `json:"userCount"`
	RoleCount        int64               `json:"roleCount"`
	MenuCount        int64               `json:"menuCount"`
	PermissionCount  int64               `json:"permissionCount"`
	OnlineSessions   int64               `json:"onlineSessions"`
	Files            int64               `json:"files"`
	Messages         int64               `json:"messages"`
	PublishedNotices int64               `json:"publishedNotices"`
	OutboxPending    int64               `json:"outboxPending"`
	OutboxDead       int64               `json:"outboxDead"`
	Range            string              `json:"range"`
	LoginTrend       []DashboardPoint    `json:"loginTrend"`
	LoginTotal       int64               `json:"loginTotal"`
	LoginPrevious    int64               `json:"loginPreviousTotal"`
	OperationTrend   []DashboardPoint    `json:"operationTrend"`
	OperationTotal   int64               `json:"operationTotal"`
	OperationSuccess float64             `json:"operationSuccessRate"`
	RoleDistribution []RoleDistribution  `json:"roleDistribution"`
	RecentLogins     []DashboardActivity `json:"recentLogins"`
	RecentActivity   []DashboardActivity `json:"recentActivity"`
}

type DashboardPoint struct {
	Label  string `json:"label"`
	Value  int64  `json:"value"`
	Failed int64  `json:"failed"`
}

type RoleDistribution struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

type DashboardActivity struct {
	Who        string    `json:"who"`
	Target     string    `json:"target"`
	Outcome    string    `json:"outcome"`
	OccurredAt time.Time `json:"occurredAt"`
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

func (s *Service) Dashboard(ctx context.Context, requestedRange string) (Dashboard, error) {
	dashboardRange := normalizeDashboardRange(requestedRange)
	var dashboard Dashboard
	dashboard.Range = dashboardRange
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
	var err error
	dashboard.LoginTrend, err = s.loginTrend(ctx, dashboardRange)
	if err != nil {
		return Dashboard{}, err
	}
	dashboard.LoginTotal, dashboard.LoginPrevious, err = s.loginTotals(ctx, dashboardRange)
	if err != nil {
		return Dashboard{}, err
	}
	dashboard.OperationTrend, err = s.operationTrend(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	dashboard.OperationTotal, dashboard.OperationSuccess, err = s.operationTotals(ctx, dashboardRange)
	if err != nil {
		return Dashboard{}, err
	}
	dashboard.RoleDistribution, err = s.roleDistribution(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	dashboard.RecentLogins, err = s.recentLogins(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	dashboard.RecentActivity, err = s.recentActivity(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	return dashboard, nil
}

func normalizeDashboardRange(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "7d":
		return "7d"
	case "30d":
		return "30d"
	default:
		return "24h"
	}
}

func (s *Service) loginTrend(ctx context.Context, dashboardRange string) ([]DashboardPoint, error) {
	if dashboardRange == "24h" {
		return s.queryDashboardPoints(ctx, `
			WITH buckets AS (
				SELECT generate_series(
					date_trunc('hour', now() AT TIME ZONE 'UTC') - interval '23 hours',
					date_trunc('hour', now() AT TIME ZONE 'UTC'),
					interval '1 hour'
				) AS bucket
			)
			SELECT to_char(bucket, 'HH24:00'), count(event.id)::bigint, 0::bigint
			FROM buckets
			LEFT JOIN audit_login_events event
				ON date_trunc('hour', event.occurred_at AT TIME ZONE 'UTC') = bucket
			GROUP BY bucket
			ORDER BY bucket`)
	}
	days := 7
	if dashboardRange == "30d" {
		days = 30
	}
	return s.queryDashboardPoints(ctx, `
		WITH buckets AS (
			SELECT generate_series(
				current_date - ($1::int - 1),
				current_date,
				interval '1 day'
			)::date AS bucket
		)
		SELECT to_char(bucket, 'MM/DD'), count(event.id)::bigint, 0::bigint
		FROM buckets
		LEFT JOIN audit_login_events event
			ON (event.occurred_at AT TIME ZONE 'UTC')::date = bucket
		GROUP BY bucket
		ORDER BY bucket`, days)
}

func (s *Service) loginTotals(ctx context.Context, dashboardRange string) (int64, int64, error) {
	selected, previous := dashboardIntervals(dashboardRange)
	var current, prior int64
	err := s.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE occurred_at >= now() - $1::interval),
			count(*) FILTER (WHERE occurred_at >= now() - $2::interval AND occurred_at < now() - $1::interval)
		FROM audit_login_events`, selected, previous).Scan(&current, &prior)
	return current, prior, err
}

func (s *Service) operationTrend(ctx context.Context) ([]DashboardPoint, error) {
	return s.queryDashboardPoints(ctx, `
		WITH buckets AS (
			SELECT generate_series(
				current_date - 6,
				current_date,
				interval '1 day'
			)::date AS bucket
		)
		SELECT to_char(bucket, 'MM/DD'),
			count(event.id)::bigint,
			count(event.id) FILTER (WHERE event.outcome <> 'success')::bigint
		FROM buckets
		LEFT JOIN audit_events event
			ON (event.occurred_at AT TIME ZONE 'UTC')::date = bucket
		GROUP BY bucket
		ORDER BY bucket`)
}

func (s *Service) operationTotals(ctx context.Context, dashboardRange string) (int64, float64, error) {
	selected, _ := dashboardIntervals(dashboardRange)
	var total, success int64
	err := s.pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE outcome = 'success')
		FROM audit_events
		WHERE occurred_at >= now() - $1::interval`, selected).Scan(&total, &success)
	if err != nil {
		return 0, 0, err
	}
	if total == 0 {
		return 0, 0, nil
	}
	return total, math.Round(float64(success)/float64(total)*1000) / 10, nil
}

func (s *Service) roleDistribution(ctx context.Context) ([]RoleDistribution, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT role.name, count(actor.id)::bigint
		FROM access_roles role
		LEFT JOIN access_user_roles user_role ON user_role.role_id = role.id
		LEFT JOIN identity_users actor ON actor.id = user_role.user_id AND actor.deleted_at IS NULL
		WHERE role.deleted_at IS NULL
		GROUP BY role.id, role.name
		ORDER BY count(actor.id) DESC, role.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RoleDistribution, 0)
	for rows.Next() {
		var item RoleDistribution
		if err := rows.Scan(&item.Label, &item.Value); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) recentLogins(ctx context.Context) ([]DashboardActivity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT username, COALESCE(NULLIF(client_type, ''), 'password'), outcome, occurred_at
		FROM audit_login_events
		ORDER BY occurred_at DESC
		LIMIT 6`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DashboardActivity, 0, 6)
	for rows.Next() {
		var item DashboardActivity
		if err := rows.Scan(&item.Who, &item.Target, &item.Outcome, &item.OccurredAt); err != nil {
			return nil, err
		}
		item.OccurredAt = item.OccurredAt.UTC()
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) recentActivity(ctx context.Context) ([]DashboardActivity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT actor_name, action || ' · ' || resource_type, outcome, occurred_at
		FROM audit_events
		ORDER BY occurred_at DESC
		LIMIT 6`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DashboardActivity, 0, 6)
	for rows.Next() {
		var item DashboardActivity
		if err := rows.Scan(&item.Who, &item.Target, &item.Outcome, &item.OccurredAt); err != nil {
			return nil, err
		}
		item.OccurredAt = item.OccurredAt.UTC()
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) queryDashboardPoints(ctx context.Context, sql string, args ...any) ([]DashboardPoint, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DashboardPoint, 0)
	for rows.Next() {
		var item DashboardPoint
		if err := rows.Scan(&item.Label, &item.Value, &item.Failed); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func dashboardIntervals(dashboardRange string) (string, string) {
	switch dashboardRange {
	case "7d":
		return "7 days", "14 days"
	case "30d":
		return "30 days", "60 days"
	default:
		return "24 hours", "48 hours"
	}
}
