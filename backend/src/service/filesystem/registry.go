package filesystem

import (
	"context"
	"sync"

	"github.com/dianlight/srat/dto"
	"gitlab.com/tozd/go/errors"
)

// Registry manages all filesystem adapters
type Registry struct {
	adapters map[string]FilesystemAdapter
	mu       sync.RWMutex
}

// NewRegistry creates a new filesystem adapter registry with all supported filesystems
func NewRegistry() *Registry {
	registry := &Registry{
		adapters: make(map[string]FilesystemAdapter),
	}

	// Register all supported filesystem adapters
	registry.Register(NewExt4Adapter())
	registry.Register(NewVfatAdapter())
	registry.Register(NewNtfsAdapter())
	registry.Register(NewBtrfsAdapter())
	registry.Register(NewXfsAdapter())
	registry.Register(NewExfatAdapter())
	registry.Register(NewF2fsAdapter())
	registry.Register(NewGfs2Adapter())
	registry.Register(NewHfsplusAdapter())
	registry.Register(NewReiserfsAdapter())
	registry.Register(NewApfsAdapter())

	return registry
}

// Register adds a filesystem adapter to the registry
func (r *Registry) Register(adapter FilesystemAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[adapter.GetName()] = adapter
}

// Get retrieves a filesystem adapter by name
func (r *Registry) Get(fsType string) (FilesystemAdapter, errors.E) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	adapter, ok := r.adapters[fsType]
	if !ok {
		return nil, errors.Errorf("unsupported filesystem type: %s", fsType)
	}
	
	return adapter, nil
}

// GetAll returns all registered filesystem adapters
func (r *Registry) GetAll() []FilesystemAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	adapters := make([]FilesystemAdapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}
	
	return adapters
}

// GetSupportedFilesystems returns information about all supported filesystems
func (r *Registry) GetSupportedFilesystems(ctx context.Context) (map[string]dto.FilesystemSupport, errors.E) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[string]dto.FilesystemSupport)
	
	for name, adapter := range r.adapters {
		support, err := adapter.IsSupported(ctx)
		if err != nil {
			return nil, errors.WithDetails(err, "Filesystem", name)
		}
		result[name] = support
	}
	
	return result, nil
}

// ListSupportedTypes returns a list of all supported filesystem type names
func (r *Registry) ListSupportedTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	types := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		types = append(types, name)
	}
	
	return types
}
