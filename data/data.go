package data

import "embed"

// FS holds every JSON fixture in this directory.
//
//go:embed *.json
var FS embed.FS
