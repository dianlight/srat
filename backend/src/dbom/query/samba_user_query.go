package query

type SambaUserQuery[T any] interface {
	// SELECT * FROM @@table WHERE is_admin = true AND deleted_at IS NULL LIMIT 1
	GetAdmin() (T, error)
	// UPDATE @@table SET deleted_at=DATETIME('now') WHERE username = @username AND is_admin = false AND deleted_at IS NULL RETURNING 1
	DeleteByName(username string) (int, error)
	// UPDATE @@table SET username = @newname WHERE username = @oldname AND is_admin = true RETURNING *
	RenameUser(oldname, newname string) (*T, error)
}
