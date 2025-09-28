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
