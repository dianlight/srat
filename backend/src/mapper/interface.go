package mapper

import "context"

type MappableTo interface {
	To(ctx context.Context, dst any) (bool, error)
}

type MappableFrom interface {
	From(ctx context.Context, src any) (bool, error)
}
