package tempiogo

import (
	"bytes"
	"log"
	"os"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
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

// RenderTemplateBuffer renders a template from a byte slice with the provided configuration.
//
// Parameters:
//   - config: A pointer to a map containing key-value pairs for template rendering.
//   - templateData: A byte slice containing the template data to be rendered.
//
// Returns:
//   - []byte: The rendered template as a byte slice.
//   - error: An error if there's an issue during template parsing or rendering.
func RenderTemplateBuffer(config *map[string]interface{}, templateData []byte) ([]byte, error) {
	buf := &bytes.Buffer{}

	// generate template
	coreTemplate := template.New("tempIO").Funcs(sprig.TxtFuncMap())
	template.Must(coreTemplate.Parse(string(templateData)))

	// render
	err := coreTemplate.Execute(buf, *config)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
