package application

var defaultStoragePath string

func SetDefaultStoragePath(path string) {
	defaultStoragePath = path
}

func getDefaultStoragePath() string {
	return defaultStoragePath
}
