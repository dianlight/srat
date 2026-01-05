package dto

//go:generate go tool goenums smart_test_types.go

// SmartTestType represents the type of SMART test to execute
type smartTestType int

const (
	smartTestTypeShort      smartTestType = iota // "short"
	smartTestTypeLong                            // "long"
	smartTestTypeConveyance                      // "conveyance"
)
