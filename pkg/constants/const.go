package constants

const (
	KokerHomePath        = "/var/lib/koker"
	KokerTempPath        = KokerHomePath + "/tmp"
	KokerImagesPath      = KokerHomePath + "/images"
	KokerContainersPath  = KokerHomePath + "/containers"
	KokerNetNsPath       = KokerHomePath + "/netns"
	KokerBridgeName      = "koker0"
	KokerBridgeDefaultIP = "172.69.0.1/16"
)