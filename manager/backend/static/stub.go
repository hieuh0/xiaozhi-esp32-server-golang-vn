//go:build !embed_ui

package static

import "embed"

// FS is empty when embed_ui is not enabled; frontend static files are not mounted in development
var FS = embed.FS{}
