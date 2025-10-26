package dto

//go:generate go tool goenums hdidle_command.go
type hdidleCommand int

const (
	scsiCommand hdidleCommand = iota // "scsi"
	ataCommand                       // "ata"
)
