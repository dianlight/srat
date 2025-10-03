package tempio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplateBuffer(t *testing.T) {
	data := map[string]interface{}{"Name": "SRAT"}
	templateContent := []byte("Hello {{ .Name }}")

	rendered, err := RenderTemplateBuffer(&data, templateContent)

	require.NoError(t, err)
	assert.Equal(t, "Hello SRAT", string(rendered))
}

func TestRenderTemplateBufferWithTemplateError(t *testing.T) {
	data := map[string]interface{}{}
	templateContent := []byte("Hello {{ .Missing }")

	_, err := RenderTemplateBuffer(&data, templateContent)

	require.Error(t, err)
}

func TestRenderTemplateFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.tpl")

	require.NoError(t, os.WriteFile(filePath, []byte("Value: {{ .Value }}"), 0o600))

	data := map[string]interface{}{"Value": "42"}

	rendered, err := RenderTemplateFile(&data, filePath)

	require.NoError(t, err)
	assert.Equal(t, "Value: 42", string(rendered))
}

func TestRenderTemplateBufferWithSprigFunctions(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		template string
		expected string
	}{
		{
			name:     "upper function",
			data:     map[string]interface{}{"text": "hello"},
			template: "{{ .text | upper }}",
			expected: "HELLO",
		},
		{
			name:     "lower function",
			data:     map[string]interface{}{"text": "WORLD"},
			template: "{{ .text | lower }}",
			expected: "world",
		},
		{
			name:     "trim function",
			data:     map[string]interface{}{"text": "  spaces  "},
			template: "{{ .text | trim }}",
			expected: "spaces",
		},
		{
			name:     "default function",
			data:     map[string]interface{}{},
			template: "{{ .missing | default \"default-value\" }}",
			expected: "default-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := RenderTemplateBuffer(&tt.data, []byte(tt.template))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(rendered))
		})
	}
}

func TestRenderTemplateBufferWithComplexData(t *testing.T) {
	data := map[string]interface{}{
		"users": []map[string]string{
			{"name": "Alice", "role": "admin"},
			{"name": "Bob", "role": "user"},
		},
	}
	template := `{{- range .users }}
{{ .name }}: {{ .role }}
{{- end }}`

	rendered, err := RenderTemplateBuffer(&data, []byte(template))
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "Alice: admin")
	assert.Contains(t, string(rendered), "Bob: user")
}

func TestRenderTemplateBufferWithNestedMaps(t *testing.T) {
	data := map[string]interface{}{
		"config": map[string]interface{}{
			"server": map[string]interface{}{
				"host": "localhost",
				"port": 8080,
			},
		},
	}
	template := "Host: {{ .config.server.host }}, Port: {{ .config.server.port }}"

	rendered, err := RenderTemplateBuffer(&data, []byte(template))
	require.NoError(t, err)
	assert.Equal(t, "Host: localhost, Port: 8080", string(rendered))
}

func TestRenderTemplateBufferWithConditionals(t *testing.T) {
	data := map[string]interface{}{
		"enabled": true,
		"debug":   false,
	}
	template := `{{- if .enabled }}
Enabled: yes
{{- end }}
{{- if .debug }}
Debug: yes
{{- else }}
Debug: no
{{- end }}`

	rendered, err := RenderTemplateBuffer(&data, []byte(template))
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "Enabled: yes")
	assert.Contains(t, string(rendered), "Debug: no")
}

func TestRenderTemplateBufferWithNumbers(t *testing.T) {
	data := map[string]interface{}{
		"count":  42,
		"price":  19.99,
		"active": true,
	}
	template := "Count: {{ .count }}, Price: {{ .price }}, Active: {{ .active }}"

	rendered, err := RenderTemplateBuffer(&data, []byte(template))
	require.NoError(t, err)
	assert.Equal(t, "Count: 42, Price: 19.99, Active: true", string(rendered))
}

func TestRenderTemplateBufferEmpty(t *testing.T) {
	data := map[string]interface{}{}
	template := []byte("Static content only")

	rendered, err := RenderTemplateBuffer(&data, template)
	require.NoError(t, err)
	assert.Equal(t, "Static content only", string(rendered))
}

func TestRenderTemplateFileNotFound(t *testing.T) {
	// Skip this test as RenderTemplateFile uses log.Fatalf which exits the process
	// This is a known limitation of the current implementation
	t.Skip("RenderTemplateFile uses log.Fatalf which cannot be tested")
}

func TestRenderTemplateFileWithSprigFunctions(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sprig.tpl")

	require.NoError(t, os.WriteFile(filePath, []byte("{{ .name | upper }}"), 0o600))

	data := map[string]interface{}{"name": "test"}

	rendered, err := RenderTemplateFile(&data, filePath)
	require.NoError(t, err)
	assert.Equal(t, "TEST", string(rendered))
}

func TestRenderTemplateBufferWithArrays(t *testing.T) {
	data := map[string]interface{}{
		"items": []string{"apple", "banana", "cherry"},
	}
	template := `{{- range .items }}
- {{ . }}
{{- end }}`

	rendered, err := RenderTemplateBuffer(&data, []byte(template))
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "- apple")
	assert.Contains(t, string(rendered), "- banana")
	assert.Contains(t, string(rendered), "- cherry")
}

func TestRenderTemplateBufferWithInvalidTemplate(t *testing.T) {
	data := map[string]interface{}{}
	template := []byte("{{ unclosed template")

	_, err := RenderTemplateBuffer(&data, template)
	require.Error(t, err)
}

func TestRenderTemplateBufferWithExecuteError(t *testing.T) {
	data := map[string]interface{}{
		"value": "test",
	}
	// Template tries to call a method that doesn't exist
	template := []byte("{{ .value.NonExistentMethod }}")

	_, err := RenderTemplateBuffer(&data, template)
	require.Error(t, err)
}

