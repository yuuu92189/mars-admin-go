package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	legacyimport "mars-admin-go/internal/migration/legacy"
	"mars-admin-go/internal/platform/config"
	"mars-admin-go/internal/platform/database"
)

type report struct {
	SourceFile  string            `json:"sourceFile"`
	GeneratedAt time.Time         `json:"generatedAt"`
	TableRows   map[string]int    `json:"tableRows"`
	Tables      []string          `json:"tables"`
	Warnings    []string          `json:"warnings"`
	Imported    bool              `json:"imported"`
	Mapping     map[string]string `json:"mapping"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	source := flag.String("source", "", "legacy MySQL SQL dump")
	mysqlDSN := flag.String("mysql-dsn", "", "legacy MySQL DSN for live import, e.g. user:pass@tcp(host:3306)/mars-system?parseTime=true&loc=UTC")
	targetURL := flag.String("target-database-url", "", "target PostgreSQL URL; defaults to DATABASE_URL")
	legacyUploadRoot := flag.String("legacy-upload-root", "", "legacy upload directory used to copy local file objects")
	targetUploadRoot := flag.String("target-upload-root", "", "target upload directory used by the local storage adapter")
	out := flag.String("report", "docs/migration/import-report.json", "report output path")
	dryRun := flag.Bool("dry-run", true, "analyze a SQL dump without writing data")
	importData := flag.Bool("import", false, "run the reviewed core data import into PostgreSQL")
	flag.Parse()
	if *importData || !*dryRun {
		if *mysqlDSN == "" {
			return fmt.Errorf("-mysql-dsn is required for a live import")
		}
		target := *targetURL
		if target == "" {
			target = os.Getenv("DATABASE_URL")
		}
		if target == "" {
			return fmt.Errorf("-target-database-url or DATABASE_URL is required")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		pool, err := database.Open(ctx, config.DatabaseConfig{
			URL: target, MaxConnections: 10, MinConnections: 1, ConnectionTimeout: 10 * time.Second,
		})
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := database.RunMigrations(ctx, pool, "up"); err != nil {
			return err
		}
		sourceDescription := *mysqlDSN
		if at := strings.Index(sourceDescription, "@"); at >= 0 {
			sourceDescription = "mysql://***@" + sourceDescription[at+1:]
		}
		result, err := legacyimport.Run(ctx, legacyimport.Options{
			MySQLDSN: *mysqlDSN, Postgres: pool, SourceDescription: sourceDescription, ReportPath: *out,
			LegacyUploadRoot: *legacyUploadRoot, TargetUploadRoot: *targetUploadRoot,
		})
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
		return err
	}
	if *source == "" {
		return fmt.Errorf("-source is required")
	}
	file, err := os.Open(*source)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	result := report{
		SourceFile: *source, GeneratedAt: time.Now().UTC(), TableRows: make(map[string]int),
		Mapping: map[string]string{
			"sys_user": "identity_users", "sys_role": "access_roles", "sys_menu": "access_menus",
			"sys_dept": "organization_departments", "sys_dict_type": "system_dictionary_types",
			"sys_dict_data": "system_dictionary_items", "sys_file": "storage_files",
			"sys_chat_message": "messaging_messages", "sys_notice": "notification_notices",
			"sys_job": "scheduler_jobs", "flow_definition": "workflow_definitions",
		},
		Warnings: []string{
			"SQL dump contains sensitive values and must be handled in a restricted environment.",
			"Legacy integer identifiers require deterministic UUID mapping before import.",
			"Live import intentionally requires a reviewed data policy.",
		},
	}
	insertPattern := regexp.MustCompile("(?i)^INSERT INTO `([^`]+)`")
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if match := insertPattern.FindStringSubmatch(line); len(match) == 2 {
			result.TableRows[match[1]]++
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	for table := range result.TableRows {
		result.Tables = append(result.Tables, table)
	}
	sort.Strings(result.Tables)
	if err := os.MkdirAll("docs/migration", 0o750); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, encoded, 0o600); err != nil {
		return err
	}
	fmt.Printf("legacy import analysis written to %s (%d tables, %d rows)\n", *out, len(result.Tables), countRows(result.TableRows))
	return nil
}

func countRows(rows map[string]int) int {
	total := 0
	for _, count := range rows {
		total += count
	}
	return total
}
