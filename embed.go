package fitness

import "embed"

//go:embed etc/scoreboard.json
//go:embed templates/index.html
var Content embed.FS
