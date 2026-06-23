package postgres

import "embed"

// MigrationsFS holds the embedded goose migration files. It is consumed by the
// migrate subcommand (goose as a library) and by the integration tests, so both
// run exactly the migrations committed alongside the adapter.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationsDir is the directory within MigrationsFS that holds the goose
// migration files. goose reads migrations relative to this path.
const MigrationsDir = "migrations"

// Migrations returns the embedded filesystem of goose migration files.
func Migrations() embed.FS {
	return migrationsFS
}
