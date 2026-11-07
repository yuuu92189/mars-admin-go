package db

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrations returns the embedded Goose migration filesystem.
func Migrations() fs.FS {
	root, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return root
}
