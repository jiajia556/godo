package templates

import "embed"

const DEFAULT_TEMPLATE_DIR = "default"

//go:embed *
var TemplateFS embed.FS
