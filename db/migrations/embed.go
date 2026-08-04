// Package migrations embeds the reviewed SQL migration files.
package migrations

import "embed"

// Files contains every versioned migration shipped with the application.
//
//go:embed *.sql
var Files embed.FS
