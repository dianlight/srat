package dbom

import (
	"gorm.io/cli/gorm/genconfig"
)

//go:generate go tool gorm gen -i ./

var _ = genconfig.Config{
	OutPath: "g",
	/*
	   // Map Go types to helper kinds
	   FieldTypeMap: map[any]any{
	     sql.NullTime{}: field.Time{},
	   },

	   // Map `gen:"name"` tags to helper kinds
	   FieldNameMap: map[string]any{
	     "json": JSON{}, // use a custom JSON helper where fields are tagged `gen:"json"`
	   },

	   // Narrow what gets generated (patterns or type literals)
	   IncludeInterfaces: []any{"Query*", models.Query(nil)},
	   IncludeStructs:    []any{"User", "Account*", models.User{}},
	*/
	IncludeInterfaces: []any{"*Query"},
	IncludeStructs:    []any{HDIdleDevice{}, MountPointPath{}, ExportedShare{}, SambaUser{}, Property{}},
}
