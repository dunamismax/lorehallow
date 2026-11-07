// Package web holds lorehallow.com's templates, stylesheet and fonts,
// embedded into cmd/lorehallow-web (ADR 0011). The generator itself is in
// internal/site.
package web

import "embed"

// Files is templates/ and static/.
//
//go:embed templates static
var Files embed.FS
