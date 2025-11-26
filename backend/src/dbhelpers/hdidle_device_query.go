package dbhelpers

// HDIdleDeviceQuery defines the methods for the HDIdleDevice query.
type HDIdleDeviceQuery[T any] interface {
	// SELECT * FROM @@table
	All() ([]*T, error)
	// SELECT * FROM @@table WHERE device_path=@path LIMIT 1
	LoadByPath(path string) (T, error)
}
