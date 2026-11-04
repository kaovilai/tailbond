// Package web holds the single-page UI, embedded into tailbondd.
package web

import "embed"

//go:embed index.html app.js style.css tokens.css icon.svg favicon.svg wordmark-dark.svg wordmark-light.svg landing.js
var FS embed.FS
