// Package web embeds the HTML templates and static assets (CSS/JS) so the
// server is a single self-contained binary.
package web

import "embed"

//go:embed templates/*.html
var Templates embed.FS

//go:embed static
var Static embed.FS
