package env

var Config struct {
	Provider string
	Region   string
}

type VersionInfo struct {
	BuildVersion string
	Commit       string
}

var BuildVersion string
var Commit string
