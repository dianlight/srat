package tempiogo

import (
	"bytes"
	"log"
	"os"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
)

func RenderTemplateFile(config *map[string]interface{}, file string) ([]byte, error) {
	// read Template
	templateFile, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("Cant read template file %s - %s", file, err)
		return nil, err
	}

	return RenderTemplateBuffer(config, templateFile)
}

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
