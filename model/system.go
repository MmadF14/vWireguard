package model

// System represents system information
type System struct {
	Version     string
	GitCommit   string
	GitRef      string
	BuildTime   string
	BasePath    string
	LoginDisabled bool
} 