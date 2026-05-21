package web

import "embed"

// Files contains the HTML templates and static assets used by the web UI.
//
//go:embed templates/*.html static/*
var Files embed.FS
