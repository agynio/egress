package migrations

import "embed"

// Files contains SQL migrations for the egress rules database.
//
//go:embed *.sql
var Files embed.FS
