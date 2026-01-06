package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service"
	"gitlab.com/tozd/go/errors" // Using aliased tozd/go/errors
)

type ShareHandler struct {
	apiContext *dto.ContextState
	//dirtyservice service.DirtyDataServiceInterface
	shareService service.ShareServiceInterface
}

func NewShareHandler(apiContext *dto.ContextState,
	//dirtyService service.DirtyDataServiceInterface,
	shareService service.ShareServiceInterface,
) *ShareHandler {
	p := new(ShareHandler)
	p.apiContext = apiContext
	//p.dirtyservice = dirtyService
	p.shareService = shareService
	return p
}

func (self *ShareHandler) RegisterShareHandler(api huma.API) {
	huma.Get(api, "/shares", self.ListShares, huma.OperationTags("share"))
	huma.Get(api, "/share/{share_name}", self.GetShare, huma.OperationTags("share"))
	huma.Post(api, "/share", self.CreateShare, huma.OperationTags("share"))
	huma.Put(api, "/share/{share_name}", self.UpdateShare, huma.OperationTags("share"))
	huma.Delete(api, "/share/{share_name}", self.DeleteShare, huma.OperationTags("share"))
	huma.Put(api, "/share/{share_name}/disable", self.DisableShare, huma.OperationTags("share"))
	huma.Put(api, "/share/{share_name}/enable", self.EnableShare, huma.OperationTags("share"))
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
	shares, err := self.shareService.ListShares()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list shares")
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
	share, err := self.shareService.GetShare(input.ShareName)
	if err != nil {
		if errors.Is(err, dto.ErrorShareNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, errors.Wrapf(err, "failed to get share %s", input.ShareName)
	}
	return &struct{ Body dto.SharedResource }{Body: *share}, nil
}

type MountPointDataNoShare struct {
	_ struct{} `json:"-" additionalProperties:"true"`
	dto.MountPointData
	Share *dto.SharedResource `json:"-" read-only:"true"` // Shares that are mounted on this mount point.
}

type SharedResourcePostData struct {
	_ struct{} `json:"-" additionalProperties:"true"`
	dto.SharedResource
	MountPointData *MountPointDataNoShare `json:"-" read-only:"true"` // Shares that are mounted on this mount point.
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
	Body SharedResourcePostData `required:"true"`
}) (*struct {
	Status int
	Body   dto.SharedResource
}, error) {
	createdShare, err := self.shareService.CreateShare(input.Body.SharedResource)
	if err != nil {
		if errors.Is(err, dto.ErrorShareAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		slog.ErrorContext(ctx, "Failed to create share", "share_name", input.Body.Name, "error", err)
		return nil, errors.Wrapf(err, "failed to create share %s", input.Body.Name)
	}

	return &struct {
		Status int
		Body   dto.SharedResource
	}{Status: http.StatusCreated, Body: *createdShare}, nil
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
	ShareName string                 `path:"share_name" maxLength:"128" example:"world" doc:"Name of the share"`
	Body      SharedResourcePostData `required:"true"`
}) (*struct{ Body dto.SharedResource }, error) {
	updatedShare, err := self.shareService.UpdateShare(input.ShareName, input.Body.SharedResource)
	if err != nil {
		if errors.Is(err, dto.ErrorShareNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, dto.ErrorShareAlreadyExists) {
			return nil, huma.Error409Conflict(err.Error())
		}
		slog.ErrorContext(ctx, "Failed to update share", "share_name", input.ShareName, "error", err)
		return nil, errors.Wrapf(err, "failed to update share %s", input.ShareName)
	}

	return &struct{ Body dto.SharedResource }{Body: *updatedShare}, nil
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
	ShareName string `path:"share_name" maxLength:"128" example:"world" doc:"Name of the share"`
}) (*struct{}, error) {
	err := self.shareService.DeleteShare(input.ShareName)
	if err != nil {
		if errors.Is(err, dto.ErrorShareNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, errors.Wrapf(err, "failed to delete share %s", input.ShareName)
	}

	return &struct{}{}, nil
}

// DisableShare disables a shared resource.
func (self *ShareHandler) DisableShare(ctx context.Context, input *struct {
	ShareName string `path:"share_name" maxLength:"128" doc:"Name of the share to disable"`
}) (*struct{ Body dto.SharedResource }, error) {
	disabledShare, err := self.shareService.DisableShare(input.ShareName)
	if err != nil {
		if errors.Is(err, dto.ErrorShareNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, errors.Wrapf(err, "failed to disable share %s", input.ShareName)
	}

	return &struct{ Body dto.SharedResource }{Body: *disabledShare}, nil
}

// EnableShare enables a shared resource.
func (self *ShareHandler) EnableShare(ctx context.Context, input *struct {
	ShareName string `path:"share_name" maxLength:"128" doc:"Name of the share to enable"`
}) (*struct{ Body dto.SharedResource }, error) {
	enabledShare, err := self.shareService.EnableShare(input.ShareName)
	if err != nil {
		if errors.Is(err, dto.ErrorShareNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, errors.Wrapf(err, "failed to enable share %s", input.ShareName)
	}

	return &struct{ Body dto.SharedResource }{Body: *enabledShare}, nil
}
