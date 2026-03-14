package dto

// AppConfigSchemaField describes one configurable app option.
type AppConfigSchemaField struct {
	Name        string `json:"name"`
	Constraint  string `json:"constraint"`
	Description string `json:"description,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// AppConfigSchema describes app configuration schema metadata.
type AppConfigSchema struct {
	Description     string                 `json:"description,omitempty"`
	LongDescription string                 `json:"long_description,omitempty"`
	RequiresRestart bool                   `json:"requires_restart"`
	Fields          []AppConfigSchemaField `json:"fields"`
}

// AppConfigData contains current app configuration values and rendered runtime config.
type AppConfigData struct {
	Options         map[string]any `json:"options"`
	RuntimeConfig   map[string]any `json:"runtime_config"`
	RequiresRestart bool           `json:"requires_restart"`
}

// AppConfigUpdateRequest is used to update app configuration options.
type AppConfigUpdateRequest struct {
	Options map[string]any `json:"options"`
}
