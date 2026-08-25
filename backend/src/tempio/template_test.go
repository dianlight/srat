package tempio

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplateBuffer(t *testing.T) {
	data := map[string]any{"Name": "SRAT"}
	templateContent := []byte("Hello {{ .Name }}")

	rendered, err := RenderTemplateBuffer(&data, templateContent)

	require.NoError(t, err)
	assert.Equal(t, "Hello SRAT", string(rendered))
}

func TestRenderTemplateBufferWithTemplateError(t *testing.T) {
	data := map[string]any{}
	templateContent := []byte("Hello {{ .Missing }")

	_, err := RenderTemplateBuffer(&data, templateContent)

	require.Error(t, err)
}

func TestRenderTemplateFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.tpl")

	require.NoError(t, os.WriteFile(filePath, []byte("Value: {{ .Value }}"), 0o600))

	data := map[string]any{"Value": "42"}

	rendered, err := RenderTemplateFile(&data, filePath)

	require.NoError(t, err)
	assert.Equal(t, "Value: 42", string(rendered))
}

func TestRenderTemplateBufferWithSprigFunctions(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		template string
		expected string
	}{
		{
			name:     "upper function",
			data:     map[string]any{"text": "hello"},
			template: "{{ .text | upper }}",
			expected: "HELLO",
		},
		{
			name:     "lower function",
			data:     map[string]any{"text": "WORLD"},
			template: "{{ .text | lower }}",
			expected: "world",
		},
		{
			name:     "trim function",
			data:     map[string]any{"text": "  spaces  "},
			template: "{{ .text | trim }}",
			expected: "spaces",
		},
		{
			name:     "default function",
			data:     map[string]any{},
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
	data := map[string]any{
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
	data := map[string]any{
		"config": map[string]any{
			"server": map[string]any{
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
	data := map[string]any{
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
	data := map[string]any{
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
	data := map[string]any{}
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

	data := map[string]any{"name": "test"}

	rendered, err := RenderTemplateFile(&data, filePath)
	require.NoError(t, err)
	assert.Equal(t, "TEST", string(rendered))
}

func TestRenderTemplateBufferWithArrays(t *testing.T) {
	data := map[string]any{
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
	data := map[string]any{}
	template := []byte("{{ unclosed template")

	_, err := RenderTemplateBuffer(&data, template)
	require.Error(t, err)
}

func TestRenderTemplateBufferWithExecuteError(t *testing.T) {
	data := map[string]any{
		"value": "test",
	}
	// Template tries to call a method that doesn't exist
	template := []byte("{{ .value.NonExistentMethod }}")

	_, err := RenderTemplateBuffer(&data, template)
	require.Error(t, err)
}
func TestRenderTemplateBufferWithAllowGuestEnabled(t *testing.T) {
	data := map[string]any{
		"allow_guest": true,
	}
	template := []byte(`{{if .allow_guest -}}
guest account = nobody
map to guest = Bad User
{{- end }}`)

	rendered, err := RenderTemplateBuffer(&data, []byte(template))
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "guest account = nobody")
	assert.Contains(t, string(rendered), "map to guest = Bad User")
}

func TestRenderTemplateBufferWithAllowGuestDisabled(t *testing.T) {
	data := map[string]any{
		"allow_guest": false,
	}
	template := []byte(`{{if .allow_guest -}}
guest account = nobody
map to guest = Bad User
{{- end }}`)

	rendered, err := RenderTemplateBuffer(&data, []byte(template))
	require.NoError(t, err)
	assert.NotContains(t, string(rendered), "guest account")
}

// loadSmbTemplate loads the production smb.gtpl template used to render smb.conf.
func loadSmbTemplate(t *testing.T) []byte {
	t.Helper()
	templateData, err := os.ReadFile("../templates/smb.gtpl")
	require.NoError(t, err)
	return templateData
}

// smbConfigForShare builds a root template context that mirrors the output of
// config.Config.ConfigToMap(): a JSON marshal/unmarshal round trip produces
// snake_case map keys and []any slices. Only the keys required for the
// [global] section and the share loop are populated; the rest fall back to
// template defaults.
func smbConfigForShare(share map[string]any) *map[string]any {
	return &map[string]any{
		"hostname":      "test-host",
		"workgroup":     "WORKGROUP",
		"username":      "admin",
		"log_level":     "fatal",
		"samba_version": "",
		"interfaces":    []any{"lo"},
		"allow_hosts":   []any{},
		"medialibrary":  map[string]any{"enable": false},
		"shares": map[string]any{
			"TEST_SHARE": share,
		},
	}
}

// extractValidUsersLine returns the "valid users = ..." line rendered for the
// TEST_SHARE section, split into whitespace-separated fields.
func extractValidUsersLine(t *testing.T, rendered []byte) []string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^\s*valid users =.*$`)
	line := re.FindString(string(rendered))
	require.NotEmpty(t, line, "valid users line not found in rendered config:\n%s", string(rendered))
	return strings.Fields(line)
}

// TestSmbTemplateValidUsersDeduplication verifies issue #991: the same user
// must never appear twice in the generated "valid users" directive, whether
// duplicated inside a single list or present in both users and ro_users.
// Order is preserved: rw users first, then ro users.
func TestSmbTemplateValidUsersDeduplication(t *testing.T) {
	tests := []struct {
		name      string
		users     []any
		roUsers   []any
		wantUsers []string
	}{
		{
			name:      "duplicate user within users list",
			users:     []any{"admin", "admin"},
			roUsers:   []any{},
			wantUsers: []string{"admin"},
		},
		{
			name:      "same user in users and ro_users",
			users:     []any{"alice", "bob"},
			roUsers:   []any{"bob", "carol"},
			wantUsers: []string{"alice", "bob", "carol"},
		},
		{
			name:      "empty users falls back to global username",
			users:     nil,
			roUsers:   []any{},
			wantUsers: []string{"admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := smbConfigForShare(map[string]any{
				"name":     "TEST_SHARE",
				"path":     "/test",
				"fs":       "native",
				"users":    tt.users,
				"ro_users": tt.roUsers,
			})

			rendered, err := RenderTemplateBuffer(data, loadSmbTemplate(t))
			require.NoError(t, err)

			fields := extractValidUsersLine(t, rendered)
			// fields[0]="valid" fields[1]="users" fields[2]="=_ha_mount_user_"
			require.GreaterOrEqual(t, len(fields), 3)
			assert.Equal(t, "=_ha_mount_user_", fields[2], "_ha_mount_user_ prefix must be preserved")
			gotUsers := fields[3:]
			assert.Equal(t, tt.wantUsers, gotUsers,
				"valid users must contain each user exactly once, preserving rw-then-ro order")
		})
	}
}

// TestSmbTemplateGuestOkEnabled verifies issue #991: a share with guest_ok set
// (snake_case key, as produced by ConfigToMap's JSON round trip) must render
// an explicit "guest ok = yes" directive.
func TestSmbTemplateGuestOkEnabled(t *testing.T) {
	data := smbConfigForShare(map[string]any{
		"name":     "TEST_SHARE",
		"path":     "/test",
		"fs":       "native",
		"users":    []any{"admin"},
		"guest_ok": true,
	})

	rendered, err := RenderTemplateBuffer(data, loadSmbTemplate(t))
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "guest ok = yes")
}

// TestSmbTemplateGuestOkDisabled guards against regressions: without guest_ok
// no "guest ok = yes" directive may be emitted for the share.
func TestSmbTemplateGuestOkDisabled(t *testing.T) {
	data := smbConfigForShare(map[string]any{
		"name":  "TEST_SHARE",
		"path":  "/test",
		"fs":    "native",
		"users": []any{"admin"},
	})

	rendered, err := RenderTemplateBuffer(data, loadSmbTemplate(t))
	require.NoError(t, err)
	assert.NotContains(t, string(rendered), "guest ok = yes")
}

// TestSmbTemplateTimeMachineMaxSize verifies issue #991 (sibling of the
// guest_ok bug): a Time Machine share with timemachine_max_size set
// (snake_case key, as produced by ConfigToMap's JSON round trip) must render
// an explicit "fruit:time machine max size" directive.
func TestSmbTemplateTimeMachineMaxSize(t *testing.T) {
	data := smbConfigForShare(map[string]any{
		"name":                 "TEST_SHARE",
		"path":                 "/test",
		"fs":                   "native",
		"users":                []any{"admin"},
		"timemachine":          true,
		"timemachine_max_size": "2 TB",
	})

	rendered, err := RenderTemplateBuffer(data, loadSmbTemplate(t))
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "fruit:time machine max size = 2 TB")
}

// TestSmbTemplateTimeMachineMaxSizeOmitted guards against regressions: no
// size directive when timemachine_max_size is unset, and none when the share
// is not a Time Machine share even if a size is configured.
func TestSmbTemplateTimeMachineMaxSizeOmitted(t *testing.T) {
	tests := []struct {
		name  string
		share map[string]any
	}{
		{
			name: "time machine share without size",
			share: map[string]any{
				"name":        "TEST_SHARE",
				"path":        "/test",
				"fs":          "native",
				"users":       []any{"admin"},
				"timemachine": true,
			},
		},
		{
			name: "size set on non time machine share",
			share: map[string]any{
				"name":                 "TEST_SHARE",
				"path":                 "/test",
				"fs":                   "native",
				"users":                []any{"admin"},
				"timemachine":          false,
				"timemachine_max_size": "2 TB",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := smbConfigForShare(tt.share)

			rendered, err := RenderTemplateBuffer(data, loadSmbTemplate(t))
			require.NoError(t, err)
			assert.NotContains(t, string(rendered), "fruit:time machine max size")
		})
	}
}
