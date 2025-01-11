package mapper

type Mappable interface {
	To(dst any) error
	//From(dst interface{}) error
}
