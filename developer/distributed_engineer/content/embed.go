// Package content embeds authored lesson files (Markdown) so they ship inside
// the server binary. Lessons are synced into the database on boot.
package content

import "embed"

//go:embed lessons/*.md
var Lessons embed.FS

//go:embed interview/*.json
var Interview embed.FS
