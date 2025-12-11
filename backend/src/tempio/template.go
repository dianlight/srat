package tempio

import (
	"bytes"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	"gitlab.com/tozd/go/errors"
)

// RenderTemplateFile renders a template file with the provided configuration.
//
// Parameters:
//   - config: A pointer to a map containing key-value pairs for template rendering.
//   - file: The path to the template file to be rendered.
//
// Returns:
//   - []byte: The rendered template as a byte slice.
//   - error: An error if the file cannot be read or if there's an issue during rendering.
func RenderTemplateFile(config *map[string]interface{}, file string) ([]byte, error) {
	// read Template
	templateFile, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Cant read template file %s - %s", file, err)
		return nil, err
	}

	return RenderTemplateBuffer(config, templateFile)
}

// versionAtLeast checks if the Samba version string is at least the required major.minor version.
// Parameters:
//   - versionStr: The version string (e.g., "4.23.0")
//   - majorRequired: The required major version
//   - minorRequired: The required minor version
//
// Returns:
//   - bool: true if version >= majorRequired.minorRequired
func versionAtLeast(versionStr string, majorRequired, minorRequired int) bool {
	if versionStr == "" {
		return false
	}

	parts := strings.Split(versionStr, ".")
	if len(parts) < 2 {
		return false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	// Check if version >= required
	if major > majorRequired || (major == majorRequired && minor >= minorRequired) {
		return true
	}

	return false
}

// versionBetween checks if the Samba version is between minVersion and maxVersion (inclusive).
// Parameters:
//   - versionStr: The version string (e.g., "4.23.0")
//   - minMajor, minMinor: The minimum required version
//   - maxMajor, maxMinor: The maximum supported version
//
// Returns:
//   - bool: true if minVersion <= version <= maxVersion
func versionBetween(versionStr string, minMajor, minMinor, maxMajor, maxMinor int) bool {
	return versionAtLeast(versionStr, minMajor, minMinor) &&
		!versionAtLeast(versionStr, maxMajor+1, 0) ||
		(versionAtLeast(versionStr, maxMajor, 0) && !versionAtLeast(versionStr, maxMajor, maxMinor+1))
}

func cleanAscii(input string) string {
	var output strings.Builder
	for _, r := range input {
		if r >= 32 && r <= 126 {
			output.WriteRune(r)
		}
	}
	return output.String()
}

// RenderTemplateBuffer renders a template from a byte slice with the provided configuration.
//
// Parameters:
//   - config: A pointer to a map containing key-value pairs for template rendering.
//   - templateData: A byte slice containing the template data to be rendered.
//
// Returns:
//   - []byte: The rendered template as a byte slice.
//   - error: An error if there's an issue during template parsing or rendering.
func RenderTemplateBuffer(config *map[string]interface{}, templateData []byte) ([]byte, errors.E) {
	buf := &bytes.Buffer{}

	// generate template with custom functions
	funcMap := sprig.TxtFuncMap()
	funcMap["versionAtLeast"] = versionAtLeast
	funcMap["versionBetween"] = versionBetween
	funcMap["cleanAscii"] = cleanAscii

	coreTemplate := template.New("tempIO").Funcs(funcMap)
	coreTemplate, err := coreTemplate.Parse(string(templateData))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// render
	err = coreTemplate.Execute(buf, *config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return buf.Bytes(), nil
}
