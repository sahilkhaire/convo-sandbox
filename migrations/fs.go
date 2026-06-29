package migrations

import "embed"

// FS contains SQL migration files for goose.
//
//go:embed *.sql
var FS embed.FS
