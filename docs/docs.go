package docs

import "embed"

// Files contains the API reference assets embedded into the backend binary.
//
//go:embed openapi/openapi.json swagger/index.html
var Files embed.FS
