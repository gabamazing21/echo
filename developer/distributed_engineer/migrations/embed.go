// Package migrations embeds the .sql migration files so they ship inside the
// single server binary — no loose files to deploy.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
