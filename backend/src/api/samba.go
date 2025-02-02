package api

import (
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/server"
	"github.com/dianlight/srat/service"
)

type SambaHanler struct {
	apictx       *ContextState
	sambaService service.SambaServiceInterface
}

func NewSambaHanler(apictx *ContextState, sambaService service.SambaServiceInterface) *SambaHanler {
	p := new(SambaHanler)
	p.apictx = apictx
	p.sambaService = sambaService
	return p
}

func (handler *SambaHanler) Patterns() []server.RouteDetail {
	return []server.RouteDetail{
		{Pattern: "/samba/apply", Method: "PUT", Handler: handler.ApplySamba},
		{Pattern: "/samba/config", Method: "GET", Handler: handler.GetSambaConfig},
	}
}

// ApplySamba godoc
//
//	@Summary		Write the samba config and send signal ro restart
//	@Description	Write the samba config and send signal ro restart
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/samba/apply [put]
func (handler *SambaHanler) ApplySamba(w http.ResponseWriter, r *http.Request) {

	stream, err := handler.sambaService.CreateConfigStream(handler.apictx.DockerInterface, handler.apictx.DockerNet, handler.apictx.Template)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	err = handler.sambaService.StreamToFile(stream, handler.apictx.SambaConfigFile)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	err = handler.sambaService.TestSambaConfig(handler.apictx.SambaConfigFile)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	err = handler.sambaService.RestartSambaService()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	HttpJSONReponse(w, nil, nil)
}

// GetSambaConfig godoc
//
//	@Summary		Get the generated samba config
//	@Description	Get the generated samba config
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	dto.SmbConf
//	@Failure		500	{object}	ErrorResponse
//	@Router			/samba/config [get]
func (handler *SambaHanler) GetSambaConfig(w http.ResponseWriter, r *http.Request) {
	var smbConf dto.SmbConf

	stream, err := handler.sambaService.CreateConfigStream(handler.apictx.DockerInterface, handler.apictx.DockerNet, handler.apictx.Template)
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	smbConf.Data = string(*stream)
	HttpJSONReponse(w, smbConf, nil)
}
