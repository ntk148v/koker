package constants

const (
	// General
	KokerApp             = "koker"
	KokerHomePath        = "/var/lib/" + KokerApp
	KokerTempPath        = KokerHomePath + "/tmp"
	KokerImagesPath      = KokerHomePath + "/images"
	KokerContainersPath  = KokerHomePath + "/containers"
	KokerNetNsPath       = KokerHomePath + "/netns"
	KokerBridgeName      = "koker0"
	KokerBridgeIPPrefix  = "172.69."
	KokerBridgeIPCIDR    = KokerBridgeIPPrefix + "0.0/16"
	KokerBridgeDefaultIP = KokerBridgeIPPrefix + "0.1"
	KokerVirtual0Pfx     = "veth0_"
	KokerVirtual1Pfx     = "veth1_"
	KokerCtrEthName      = "eth0"

	// CGroup
	CGroupMountpoint = "/sys/fs/cgroup"
	DefaultCfsPeriod = 100000

	// Template
	ContainersTemplate = `
CONTAINER ID{{"\t\t"}}IMAGE       {{"\t\t"}}COMMAND
{{ range $container := . }}
{{ $container.id }}{{"\t"}}{{ printf "%.12s" $container.image }}{{"\t\t"}}{{ $container.cmd }}
{{ end }}
`
	ImagesTemplate = `
REPOSITORY{{"\t\t"}}TAG{{"\t\t"}}IMAGE ID
{{ range $image := . }}
{{ $image.repository }}{{"\t\t"}}{{ $image.tag }}{{"\t\t"}}{{ printf "%.12s" $image.id }}
{{ end }}
`
)
