package mapper

type Mappable[T any] interface {
	To(dst *T) error
}
