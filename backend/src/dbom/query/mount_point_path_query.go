package query

type MountPointPathQuery[T any] interface {
	// SELECT * FROM @@table
	All() ([]*T, error)
	// SELECT * FROM @@table WHERE path=@path
	//FindByPath(path string) (dbom.MountPointPath, error)
	// SELECT * FROM @@table WHERE device_id=@device
	//FindByDevice(device string) ([]*dbom.MountPointPath, error)
}
