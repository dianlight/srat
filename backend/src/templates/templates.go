package templates

import "embed"

//go:embed smb.gtpl
var Template_content embed.FS

//go:embed default_config.json
var Default_Config_content embed.FS
