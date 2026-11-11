// Package migrations embeds the goose SQL migrations so tests (and, later,
// the server) can apply them without a filesystem path.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
