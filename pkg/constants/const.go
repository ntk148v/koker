package constants

const (
	KokerApp             = "koker"
	KokerHomePath        = "/var/lib/" + KokerApp
	KokerTempPath        = KokerHomePath + "/tmp"
	KokerImagesPath      = KokerHomePath + "/images"
	KokerContainersPath  = KokerHomePath + "/containers"
	KokerNetNsPath       = KokerHomePath + "/netns"
	KokerBridgeName      = "koker0"
	KokerBridgeDefaultIP = "172.69.0.1"
	KokerVirtual0Pfx     = "veth0_"
	KokerVirtual1Pfx     = "veth1_"
)
