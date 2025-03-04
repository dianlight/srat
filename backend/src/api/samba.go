package api

import (
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
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
	fuego.Put(srv, "/samba/apply", handler.ApplySamba, option.Tags("samba"), option.Description("Write the samba config and send signal to restart"))
	fuego.Get(srv, "/samba/config", handler.GetSambaConfig, option.Tags("samba"), option.Description("Get the generated samba config"))

	return nil
}

func (handler *SambaHanler) ApplySamba(c fuego.ContextNoBody) (bool, error) {

	err := handler.sambaService.WriteSambaConfig()
	if err != nil {
		return false, err
	}

	err = handler.sambaService.TestSambaConfig()
	if err != nil {
		return false, err
	}

	err = handler.sambaService.RestartSambaService()
	if err != nil {
		return false, err
	}

	return true, nil
}

func (handler *SambaHanler) GetSambaConfig(c fuego.ContextNoBody) (*dto.SmbConf, error) {
	var smbConf dto.SmbConf

	stream, err := handler.sambaService.CreateConfigStream()
	if err != nil {
		return nil, err
	}

	smbConf.Data = string(*stream)
	return &smbConf, nil
}
