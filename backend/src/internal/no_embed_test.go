//go:build !embedallowed

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFrontendDefaultsToStaticDirectory(t *testing.T) {
	original := Frontend
	Frontend = nil
	t.Cleanup(func() { Frontend = original })

	fsys := GetFrontend()
	require.NotNil(t, Frontend)
	assert.Equal(t, "web/static", *Frontend)
	require.NotNil(t, fsys)
}

func TestGetTemplateDataReadsCustomFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "template.tpl")
	require.NoError(t, os.WriteFile(filePath, []byte("custom"), 0o600))

	original := TemplateFile
	TemplateFile = &filePath
	t.Cleanup(func() { TemplateFile = original })

	data := GetTemplateData()
	assert.Equal(t, "custom", string(data))
}

func TestGetTemplateDataDefaultsToEmbeddedPath(t *testing.T) {
	original := TemplateFile
	TemplateFile = nil
	t.Cleanup(func() { TemplateFile = original })

	_ = GetTemplateData()
	require.NotNil(t, TemplateFile)
	assert.Equal(t, "templates/smb.gtpl", *TemplateFile)
}
