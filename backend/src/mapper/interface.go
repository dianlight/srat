package mapper

type MappableTo interface {
	To(dst any) (bool, error)
}

type MappableFrom interface {
	From(src any) (bool, error)
}
