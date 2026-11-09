package web

import "embed"

// Assets travels inside the Go binary, so startup does not depend on a working directory.
//
//go:embed index.html app.js styles.css
var Assets embed.FS
