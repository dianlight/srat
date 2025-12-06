package query

// HDIdleDeviceQuery defines the methods for the HDIdleDevice query.
type HDIdleDeviceQuery[T any] interface {
	// SELECT * FROM @@table
	All() ([]*T, error)
}
