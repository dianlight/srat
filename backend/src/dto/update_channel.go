package dto

//go:generate go tool goenums -l -i update_channel.go
type updateChannel int8

const (
	None       updateChannel = iota // "Release"
	Develop                         // "Develop"
	Release                         // "None"
	Prerelease                      // "Prerelease"
)
