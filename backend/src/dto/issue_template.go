package dto

// IssueTemplateField represents a single field in the GitHub issue template
type IssueTemplateField struct {
	Type        string                 `json:"type" yaml:"type"`
	ID          string                 `json:"id" yaml:"id"`
	Attributes  IssueTemplateFieldAttr `json:"attributes" yaml:"attributes"`
	Validations *IssueTemplateValidity `json:"validations,omitempty" yaml:"validations,omitempty"`
}

// IssueTemplateFieldAttr represents the attributes of a template field
type IssueTemplateFieldAttr struct {
	Label       string   `json:"label" yaml:"label"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Placeholder string   `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
	Options     []string `json:"options,omitempty" yaml:"options,omitempty"`
	Multiple    bool     `json:"multiple,omitempty" yaml:"multiple,omitempty"`
	Render      string   `json:"render,omitempty" yaml:"render,omitempty"`
}

// IssueTemplateValidity represents validation rules for a field
type IssueTemplateValidity struct {
	Required bool `json:"required" yaml:"required"`
}

// IssueTemplate represents the parsed GitHub issue template
type IssueTemplate struct {
	Name        string               `json:"name" yaml:"name"`
	Description string               `json:"description" yaml:"description"`
	Title       string               `json:"title" yaml:"title"`
	Labels      []string             `json:"labels" yaml:"labels"`
	Body        []IssueTemplateField `json:"body" yaml:"body"`
}

// IssueTemplateResponse is the API response containing the parsed template
type IssueTemplateResponse struct {
	Template *IssueTemplate `json:"template"`
	Error    *string        `json:"error,omitempty"`
}
