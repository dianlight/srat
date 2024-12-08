package main

import (
	"log"
	"net/http"
	"os"

	tempiogo "github.com/dianlight/srat/tempio"
)

func createConfigStream() *[]byte {
	config_2 := configToMap(config)
	//log.Printf("New Config:\n\t%s", config_2)
	data := tempiogo.RenderTemplateBuffer(config_2, templateData)
	return &data
}

// ApplySamba godoc
//
//	@Summary		Write the samba config and send signal ro restart
//	@Description	Write the samba config and send signal ro restart
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		204
//	@Success		200	{object}	[]byte
//	@Failure		400	{object}	ResponseError
//	@Failure		500	{object}	ResponseError
//	@Router			/samba/apply [put]
func applySamba(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stream := createConfigStream()
	if *smbConfigFile == "" {
		// Only for DEBUG propose
		w.WriteHeader(http.StatusOK)
		w.Write(*stream)
	} else {
		err := os.WriteFile(*smbConfigFile, *stream, 0644)
		if err != nil {
			log.Fatal(err)
		}
		// TODO: Send signal to restart samba
		w.WriteHeader(http.StatusNoContent)
	}
}
