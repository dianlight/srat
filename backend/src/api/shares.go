package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/repository"
	"github.com/dianlight/srat/service"
	"gorm.io/gorm"
)

type ShareHandler struct {
	sharesQueueMutex    sync.RWMutex
	broadcaster         service.BroadcasterServiceInterface
	apiContext          *dto.ContextState
	dirtyservice        service.DirtyDataServiceInterface
	exported_share_repo repository.ExportedShareRepositoryInterface
}

func NewShareHandler(broadcaster service.BroadcasterServiceInterface,
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
	return p
}

func (self *ShareHandler) RegisterShareHandler(api huma.API) {
	huma.Get(api, "/shares", self.ListShares, huma.OperationTags("share"))
	huma.Get(api, "/share/{share_name}", self.GetShare, huma.OperationTags("share"))
	huma.Post(api, "/share", self.CreateShare, huma.OperationTags("share"))
	huma.Put(api, "/share/{share_name}", self.UpdateShare, huma.OperationTags("share"))
	huma.Delete(api, "/share/{share_name}", self.DeleteShare, huma.OperationTags("share"))
}

// ListShares retrieves a list of shared resources from the repository,
// converts them to the appropriate DTO format, and returns them.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and deadlines.
//   - input: An empty struct for future extensibility.
//
// Returns:
//   - A struct containing a slice of shared resources in the response body.
//   - An error if there is any issue retrieving or converting the shared resources.
func (self *ShareHandler) ListShares(ctx context.Context, input *struct{}) (*struct{ Body []dto.SharedResource }, error) {
	var shares []dto.SharedResource
	var dbshares []dbom.ExportedShare
	err := self.exported_share_repo.All(&dbshares)
	if err != nil {
		return nil, err
	}
	var conv converter.DtoToDbomConverterImpl
	for _, dbshare := range dbshares {
		var share dto.SharedResource
		err = conv.ExportedShareToSharedResource(dbshare, &share)
		if err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	return &struct{ Body []dto.SharedResource }{Body: shares}, nil
}

// GetShare retrieves a shared resource by its name.
//
//	@param	ctx		-	The	context	for			the	request.
//	@param	input	-	A	struct	containing	the	ShareName,	which	is	the	name	of	the	share	to	retrieve.
//
//	@return	A struct containing the shared resource in the Body field, or an error if the share is not found or another error occurs.
//
// The ShareName field in the input struct has the following tags:
// - path:"share_name": Indicates that this field is part of the URL path.
// - maxLength:"30": Specifies the maximum length of the ShareName.
// - example:"world": Provides an example value for the ShareName.
// - doc:"Name of the share": A description of the ShareName field.
//
// Possible errors:
// - huma.Error404NotFound: Returned if the share is not found.
// - Other errors may be returned if there is an issue with the database or the conversion process.
func (self *ShareHandler) GetShare(ctx context.Context, input *struct {
	ShareName string `path:"share_name" maxLength:"30" example:"world" doc:"Name of the share"`
}) (*struct{ Body dto.SharedResource }, error) {

	dbshare, err := self.exported_share_repo.FindByName(input.ShareName)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.Error404NotFound("Share not found")
	} else if err != nil {
		return nil, err
	}
	share := dto.SharedResource{}
	var conv converter.DtoToDbomConverterImpl
	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	if err != nil {
		return nil, err
	}

	return &struct{ Body dto.SharedResource }{Body: share}, nil
}

// CreateShare handles the creation of a new shared resource.
// It takes a context and an input struct containing the shared resource data.
// The function performs the following steps:
// 1. Converts the input DTO to a database object.
// 2. Ensures that the share has at least one user, adding an admin user if necessary.
// 3. Saves the share to the repository.
// 4. Converts the saved database object back to a DTO.
// 5. Marks shares as dirty and notifies the client.
//
// Parameters:
// - ctx: The context for the request.
// - input: A struct containing the shared resource data.
//
// Returns:
// - A struct containing the status code and the created shared resource DTO.
// - An error if any step in the process fails.
func (self *ShareHandler) CreateShare(ctx context.Context, input *struct {
	Body dto.SharedResource
}) (*struct {
	Status int
	Body   dto.SharedResource
}, error) {

	dbshare := &dbom.ExportedShare{
		Name: input.Body.Name,
	}
	var conv converter.DtoToDbomConverterImpl
	err := conv.SharedResourceToExportedShare(input.Body, dbshare)
	if err != nil {
		return nil, err
	}
	if len(dbshare.Users) == 0 {
		adminUser := dbom.SambaUser{}
		err = adminUser.GetAdmin()
		if err != nil {
			return nil, err
		}
		dbshare.Users = append(dbshare.Users, adminUser)
	}
	err = self.exported_share_repo.Save(dbshare)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, huma.Error409Conflict("Share already exists")
		}
		return nil, err
	}

	var share dto.SharedResource
	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	if err != nil {
		return nil, err
	}
	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()

	return &struct {
		Status int
		Body   dto.SharedResource
	}{Status: http.StatusCreated, Body: share}, nil
}

// notifyClient retrieves all exported shares from the repository, converts them to shared resources,
// and broadcasts the list of shared resources to clients. It uses a read lock to ensure thread-safe
// access to the shares queue.
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
	self.broadcaster.BroadcastMessage(shares)
}

// UpdateShare updates an existing share with the provided input data.
//
// Parameters:
//   - ctx: The context for the request.
//   - input: A struct containing the ShareName and the Body with the updated share data.
//
// Returns:
//   - A struct containing the updated share data in the Body field.
//   - An error if the update operation fails.
//
// Possible Errors:
//   - huma.Error404NotFound: If the share with the specified name is not found.
//   - huma.Error409Conflict: If there is a conflict while updating the share name.
//   - Other errors related to database operations or data conversion.
//
// This method performs the following steps:
//  1. Finds the share by name using the exported_share_repo.
//  2. Converts the input Body to the database share model.
//  3. Updates the share name if it has changed.
//  4. Saves the updated share to the database.
//  5. Converts the updated database share model back to the DTO.
//  6. Marks shares as dirty and notifies the client asynchronously.
func (self *ShareHandler) UpdateShare(ctx context.Context, input *struct {
	ShareName string `path:"share_name" maxLength:"30" example:"world" doc:"Name of the share"`
	Body      dto.SharedResource
}) (*struct{ Body dto.SharedResource }, error) {

	dbshare, err := self.exported_share_repo.FindByName(input.ShareName)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.Error404NotFound("Share not found")
	} else if err != nil {
		return nil, err
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(input.Body, dbshare)
	if err != nil {
		return nil, err
	}

	if input.ShareName != dbshare.Name {
		err = self.exported_share_repo.UpdateName(input.ShareName, dbshare.Name)
		if err != nil {
			return nil, huma.Error409Conflict("Share already exists")
		}
	}

	err = self.exported_share_repo.Save(dbshare)
	if err != nil {
		return nil, err
	}

	var share dto.SharedResource
	err = conv.ExportedShareToSharedResource(*dbshare, &share)
	if err != nil {
		return nil, err
	}

	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()
	return &struct{ Body dto.SharedResource }{Body: share}, nil
}

// DeleteShare handles the deletion of a shared resource by its name.
// It takes a context and an input struct containing the share name.
// If the share is not found, it returns a 404 error. If any other error occurs, it returns that error.
// On successful deletion, it marks the shares as dirty and notifies the client.
//
// Parameters:
//
//	ctx - The context for the request.
//	input - A struct containing the share name to be deleted.
//
// Returns:
//
//	An empty struct on success, or an error if the deletion fails.
func (self *ShareHandler) DeleteShare(ctx context.Context, input *struct {
	ShareName string `path:"share_name" maxLength:"30" example:"world" doc:"Name of the share"`
}) (*struct{}, error) {
	var share dto.SharedResource
	dbshare, err := self.exported_share_repo.FindByName(input.ShareName)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.Error404NotFound("Share not found")
	} else if err != nil {
		return nil, err
	}
	var conv converter.DtoToDbomConverterImpl
	err = conv.SharedResourceToExportedShare(share, dbshare)
	if err != nil {
		return nil, err
	}
	err = self.exported_share_repo.Delete(dbshare.Name)
	if err != nil {
		return nil, err
	}

	self.dirtyservice.SetDirtyShares()
	go self.notifyClient()
	return &struct{}{}, nil
}
