package query

type MountPointPathQuery[T any] interface {
	// SELECT * FROM @@table
	All() ([]*T, error)
}
