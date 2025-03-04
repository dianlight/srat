package api

import (
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type ShareHandler struct {
	sharesQueueMutex    sync.RWMutex
	broadcaster         service.BroadcasterServiceInterface
	apiContext          *dto.ContextState
	dirtyservice        service.DirtyDataServiceInterface
	exported_share_repo repository.ExportedShareRepositoryInterface
}

func NewShareHandler(
	broadcaster service.BroadcasterServiceInterface,
	apiContext *dto.ContextState,
	dirtyService service.DirtyDataServiceInterface,
	exported_share_repo repository.ExportedShareRepositoryInterface,
) *ShareHandler {
	p := new(ShareHandler)
	p.broadcaster = broadcaster
	p.apiContext = apiContext
	p.dirtyservice = dirtyService
	p.exported_share_repo = exported_share_repo
	p.sharesQueueMutex = sync.RWMutex{}
	broadcaster.AddOpenConnectionListener(func(broker service.BroadcasterServiceInterface) error {
		p.notifyClient()
		return nil
	})

	return p
}

func (p *ShareHandler) Routers(srv *fuego.Server) error {
	fuego.Get(srv, "/shares", p.ListShares, option.Description("List all configured shares"), option.Tags("share"))
	fuego.Get(srv, "/share/{share_name}", p.GetShare, option.Description("get share by Name"), option.Tags("share"))
	fuego.Post(srv, "/share", p.CreateShare, option.Description("create e new share"), option.Tags("share"))
	fuego.Put(srv, "/share/{share_name}", p.UpdateShare, option.Description("update e new share"), option.Tags("share"))
	fuego.Delete(srv, "/share/{share_name}", p.DeleteShare, option.Description("delere a share"), option.Tags("share"))
	return nil
}

func (self *ShareHandler) ListShares(c fuego.ContextNoBody) ([]dto.SharedResource, error) {
	var shares []dto.SharedResource
	var dbshares []dbom.ExportedShare
	err := self.exported_share_repo.All(&dbshares)
	if err != nil {
		return shares, tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl
	for _, dbshare := range dbshares {
		var share dto.SharedResource
		err = conv.ExportedShareToSharedResource(dbshare, &share)
		if err != nil {
			return shares, tracerr.Wrap(err)
		}
		shares = append(shares, share)
	}
	return shares, nil
}

func (self *ShareHandler) GetShare(c fuego.ContextNoBody) (dto.SharedResource, error) {
	shareName := c.PathParam("share_name")

	dbshare, err := self.exported_share_repo.FindByName(shareName)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.SharedResource{}, fuego.NotFoundError{
			Title: "Share not found",
		}
	} else if err != nil {
		return dto.SharedResource{}, tracerr.Wrap(err)
	}
	share := dto.SharedResource{}
	var conv converter.DtoToDbomConverterImpl
	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	if err != nil {
		return dto.SharedResource{}, tracerr.Wrap(err)
	}

	return share, nil
}

func (self *ShareHandler) CreateShare(c fuego.ContextWithBody[dto.SharedResource]) (*dto.SharedResource, error) {

	share, err := c.Body()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	dbshare := &dbom.ExportedShare{
		Name: share.Name,
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, dbshare)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	if len(dbshare.Users) == 0 {
		adminUser := dbom.SambaUser{}
		err = adminUser.GetAdmin()
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
		dbshare.Users = append(dbshare.Users, adminUser)
	}
	err = self.exported_share_repo.Save(dbshare)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fuego.ConflictError{
				Title: "Share already exists"}

		}
		return nil, tracerr.Wrap(err)
	}

	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()

	c.SetStatus(http.StatusCreated)

	return &share, nil
}

func (self *ShareHandler) notifyClient() {
	self.sharesQueueMutex.RLock()
	defer self.sharesQueueMutex.RUnlock()
	var shares []dto.SharedResource
	var dbshares = []dbom.ExportedShare{}
	err := self.exported_share_repo.All(&dbshares)
	if err != nil {
		log.Fatal(err)
		return
	}
	var conv converter.DtoToDbomConverterImpl
	for _, dbshare := range dbshares {
		var share dto.SharedResource
		err = conv.ExportedShareToSharedResource(dbshare, &share)
		if err != nil {
			log.Fatal(err)
			return
		}
		shares = append(shares, share)
	}

	var event dto.EventMessageEnvelope
	event.Event = dto.EventShare
	event.Data = shares
	self.broadcaster.BroadcastMessage(&event)
}

func (self *ShareHandler) UpdateShare(c fuego.ContextWithBody[dto.SharedResource]) (*dto.SharedResource, error) {
	share_name := c.PathParam("share_name")

	share, err := c.Body()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	dbshare, err := self.exported_share_repo.FindByName(share_name)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fuego.NotFoundError{
			Title: "Share not found",
		}
	} else if err != nil {
		return nil, tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, dbshare)
	//	err = mapper.Map(context.Background(), &dbshare, share)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	if share_name != dbshare.Name {
		err = self.exported_share_repo.UpdateName(share_name, dbshare.Name)
		if err != nil {
			return nil, tracerr.Wrap(err)
		}
	}

	err = self.exported_share_repo.Save(dbshare)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	if err != nil {
		return nil, tracerr.Wrap(err)
	}

	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()
	return &share, nil
}

func (self *ShareHandler) DeleteShare(c fuego.ContextNoBody) (bool, error) {
	share_name := c.PathParam("share_name")

	var share dto.SharedResource
	dbshare, err := self.exported_share_repo.FindByName(share_name)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return false, fuego.NotFoundError{
			Title: "Share not found",
		}
	} else if err != nil {
		return false, tracerr.Wrap(err)
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, dbshare)
	if err != nil {
		return false, tracerr.Wrap(err)
	}
	err = self.exported_share_repo.Delete(dbshare.Name)
	if err != nil {
		return false, tracerr.Wrap(err)
	}

	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()

	return true, nil
}
