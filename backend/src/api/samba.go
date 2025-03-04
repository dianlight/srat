package api

import (
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
)

type SambaHanler struct {
	apictx       *dto.ContextState
	sambaService service.SambaServiceInterface
}

func NewSambaHanler(apictx *dto.ContextState, sambaService service.SambaServiceInterface) *SambaHanler {
	p := new(SambaHanler)
	p.apictx = apictx
	p.sambaService = sambaService

	return p
}

func (handler *SambaHanler) Routers(srv *fuego.Server) error {
	fuego.PutStd(srv, "/samba/apply", handler.ApplySamba)
	fuego.GetStd(srv, "/samba/config", handler.GetSambaConfig)

	return nil
}

// ApplySamba godoc
//
//	@Summary		Write the samba config and send signal ro restart
//	@Description	Write the samba config and send signal ro restart
//	@Tags			samba
//	@Accept			json
//	@Produce		json
//	@Success		204
//	@Failure		400	{object}	dto.ErrorInfo
//	@Failure		500	{object}	dto.ErrorInfo
//	@Router			/samba/apply [put]
func (handler *SambaHanler) ApplySamba(w http.ResponseWriter, r *http.Request) {

	err := handler.sambaService.WriteSambaConfig()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	err = handler.sambaService.TestSambaConfig()
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
//	@Failure		500	{object}	dto.ErrorInfo
//	@Router			/samba/config [get]
func (handler *SambaHanler) GetSambaConfig(w http.ResponseWriter, r *http.Request) {
	var smbConf dto.SmbConf

	stream, err := handler.sambaService.CreateConfigStream()
	if err != nil {
		HttpJSONReponse(w, err, nil)
		return
	}

	smbConf.Data = string(*stream)
	HttpJSONReponse(w, smbConf, nil)
}
