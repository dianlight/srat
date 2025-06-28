package api

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
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

func (self *SambaHanler) RegisterSambaHandler(api huma.API) {
	huma.Get(api, "/samba/config", self.GetSambaConfig, huma.OperationTags("samba"))
	huma.Put(api, "/samba/apply", self.ApplySamba, huma.OperationTags("samba"))
	huma.Get(api, "/samba/status", self.GetSambaStatus, huma.OperationTags("samba"))
}

func (handler *SambaHanler) GetSambaStatus(ctx context.Context, input *struct{}) (*struct{ Body dto.SambaStatus }, error) {
	status, err := handler.sambaService.GetSambaStatus()
	if err != nil {
		return nil, err
	}
	return &struct{ Body dto.SambaStatus }{Body: *status}, nil
}

// ApplySamba applies the Samba configuration by writing, testing, and restarting the Samba service.
// It returns an error if any of the steps fail.
//
// Parameters:
//   - ctx: The context for the operation.
//   - input: A pointer to an empty struct.
//
// Returns:
//   - A pointer to an empty struct.
//   - An error if any of the steps fail.
func (handler *SambaHanler) ApplySamba(ctx context.Context, input *struct{}) (*struct{}, error) {

	err := handler.sambaService.WriteSambaConfig()
	if err != nil {
		return nil, err
	}

	err = handler.sambaService.TestSambaConfig()
	if err != nil {
		return nil, err
	}

	err = handler.sambaService.RestartSambaService()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// GetSambaConfig retrieves the Samba configuration.
// It creates a configuration stream using the sambaService and converts it to a string.
// The configuration is then returned wrapped in a struct containing the dto.SmbConf.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: An empty struct.
//
// Returns:
//   - A struct containing the Samba configuration in the Body field.
//   - An error if there is an issue creating the configuration stream.
func (handler *SambaHanler) GetSambaConfig(ctx context.Context, input *struct{}) (*struct{ Body dto.SmbConf }, error) {
	var smbConf dto.SmbConf

	stream, err := handler.sambaService.CreateConfigStream()
	if err != nil {
		return nil, err
	}

	smbConf.Data = string(*stream)

	return &struct{ Body dto.SmbConf }{Body: smbConf}, nil
}
