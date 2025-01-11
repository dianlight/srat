package mapper

type MappableTo interface {
	To(dst any) error
}

type MappableFrom interface {
	From(src any) error
}
