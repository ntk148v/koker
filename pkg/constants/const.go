package constants

const (
	KokerApp             = "koker"
	KokerHomePath        = "/var/lib/" + KokerApp
	KokerTempPath        = KokerHomePath + "/tmp"
	KokerImagesPath      = KokerHomePath + "/images"
	KokerContainersPath  = KokerHomePath + "/containers"
	KokerNetNsPath       = KokerHomePath + "/netns"
	KokerImageLayersPath = KokerImagesPath + "/layers"
	KokerBridgeName      = "koker0"
	KokerBridgeDefaultIP = "172.69.0.1"
	KokerVirtual0Pfx     = "veth0_"
	KokerVirtual1Pfx     = "veth1_"
	KokerCtrEthName      = "eth0"

	// CGroup
	CGroupPath           = "/sys/fs/cgroup"
	ReleaseAgentFilename = "notify_on_release"
	ProcsFilename        = "cgroup.procs"
	MemLimitFilename     = "memory.limit_in_bytes"
	MemswLimitFilename   = "memory.memsw.limit_in_bytes"
	CpuQuotaFilename     = "cpu.cfs_quota_us"
	CpuPeriodFilename    = "cpu.cfs_period_us"
	MaxProcessFilename   = "pids.max"
	DefaultCfsPeriod     = 100000
)
