package network

import (
	"net"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/utils"
)

// CheckBridgeUp check whether the bridge is up
func CheckBridgeUp() (bool, error) {
	log.Info().Str("bridge", constants.KokerBridgeName).
		Msg("Check default bridge is up or not")
	links, err := netlink.LinkList()
	if err == nil {
		for _, link := range links {
			if link.Type() == "bridge" && link.Attrs().Name == constants.KokerBridgeName {
				return true, nil
			}
		}
	}

	log.Error().Err(err).Msg("Unable to get list of links")
	return false, err
}

// SetupBridge sets up the default bridge interface.
// To keep things simple, assign ip 172.69.0.1 to it (yeah, I fixed it!)
// which is from the range of IPs which will also use for the containers
func SetupBridge() error {
	log.Info().Str("bridge", constants.KokerBridgeName).
		Msg("Setup default bridge")
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = constants.KokerBridgeName
	log.Debug().Str("bridge", constants.KokerBridgeName).
		Msg("Add a new link device")
	bridge := &netlink.Bridge{LinkAttrs: linkAttrs}
	if err := netlink.LinkAdd(bridge); err != nil {
		return err
	}

	addr, _ := netlink.ParseAddr(constants.KokerBridgeDefaultIP + "/16")
	log.Debug().Str("bridge", constants.KokerBridgeName).
		Msg("Add an IP address to bridge")
	netlink.AddrAdd(bridge, addr)
	log.Debug().Str("bridge", constants.KokerBridgeName).
		Msg("Bring bridge up")
	netlink.LinkSetUp(bridge)
	return nil
}

// SetupVirtEth creates and configures a virtual ethernet on host
func SetupVirtEth(conID string) error {
	veth0Name := constants.KokerVirtual0Pfx + conID[:6]
	veth1Name := constants.KokerVirtual1Pfx + conID[:6]

	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = veth0Name

	log.Debug().Str("veth0", veth0Name).
		Msg("Add a new link")
	veth0MAC, err := utils.GenMac()
	if err != nil {
		log.Error().Err(err).Str("veth0", veth0Name).
			Msg("Unable to generate MAC address for new virtual ethernet")
		return err
	}
	veth0 := &netlink.Veth{
		LinkAttrs:        linkAttrs,
		PeerName:         veth1Name,
		PeerHardwareAddr: veth0MAC,
	}
	if err := netlink.LinkAdd(veth0); err != nil {
		log.Error().Err(err).Str("veth0", veth0Name).
			Msg("Unable to add new virtual ethernet")
		return err
	}

	log.Debug().Str("veth0", veth0Name).
		Msg("Bring virtual ethernet up")
	netlink.LinkSetUp(veth0)
	log.Debug().Str("veth0", veth0Name).
		Str("bridge", constants.KokerBridgeName).
		Msg("Set default bridge as the master of new virtual ethernet")
	koker0, _ := netlink.LinkByName(constants.KokerBridgeName)
	netlink.LinkSetMaster(veth0, koker0)

	return nil
}

// SetupNetNS creates new network namespace for input container
func SetupNetNS(conID string) error {
	_ = utils.CreateDir(constants.KokerNetNsPath)
	nsMount := filepath.Join(constants.KokerNetNsPath, conID)
	if _, err := unix.Open(nsMount, unix.O_RDONLY|unix.O_CREAT|unix.O_EXCL, 0644); err != nil {
		log.Error().Err(err).Str("netnsmount", nsMount).
			Msg("Unable to open bind mount file")
		return err
	}

	fd, err := unix.Open("/proc/self/ns/net", unix.O_RDONLY, 0)
	defer unix.Close(fd)
	if err != nil {
		log.Error().Err(err).Msg("Unable to open file")
		return err
	}

	if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
		log.Error().Err(err).Msg("Unshare system call failed")
		return err
	}
	if err := unix.Mount("/proc/self/ns/net", nsMount, "bind", unix.MS_BIND, ""); err != nil {
		log.Error().Err(err).Msg("Mount system call failed")
		return err
	}
	if err := unix.Setns(fd, unix.CLONE_NEWNET); err != nil {
		log.Error().Err(err).Msg("Setns system call failed")
		return err
	}
	return nil
}

// SetupConNetInf setups container network inteface
func SetupConNetInf(conID string) error {
	nsMount := filepath.Join(constants.KokerNetNsPath, conID)

	fd, err := unix.Open(nsMount, unix.O_RDONLY, 0)
	defer unix.Close(fd)
	if err != nil {
		log.Error().Err(err).Msg("Unable to open")
		return err
	}

	// Set veth1 of new container to the new network namespace
	veth1Name := constants.KokerVirtual1Pfx + conID[:6]
	veth1, err := netlink.LinkByName(veth1Name)
	if err != nil {
		log.Error().Err(err).Str("veth1", veth1Name).
			Msg("Unable to fetch virtual ethernet")
		return err
	}

	log.Debug().Str("veth1", veth1Name).
		Msg("Set network netnamespace virtual ethernet")
	if err := netlink.LinkSetNsFd(veth1, fd); err != nil {
		log.Error().Err(err).Str("veth1", veth1Name).
			Msg("Unable to set network namespace for vritual ethernet")
		return err
	}

	if err := unix.Setns(fd, unix.CLONE_NEWNET); err != nil {
		log.Error().Err(err).Msg("Setns system call failed")
		return err
	}

	log.Debug().Str("veth1", veth1Name).
		Msg("Assign IP to virtual ethernet")
	addr, _ := netlink.ParseAddr(utils.GenIPAddress() + "/16")
	if err := netlink.AddrAdd(veth1, addr); err != nil {
		log.Error().Err(err).Str("veth1", veth1Name).
			Msg("Unable to assign IP to virtual ethernet")
		return err
	}

	log.Debug().Str("veth1", veth1Name).
		Msg("Bring up virtual ethernet")
	if err := netlink.LinkSetUp(veth1); err != nil {
		log.Error().Err(err).Str("veth1", veth1Name).
			Msg("Unable to bring up virtual ethernet")
		return err
	}

	log.Debug().Str("veth1", veth1Name).
		Msg("Add route for virtual ethernetl")
	route := netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: veth1.Attrs().Index,
		Gw:        net.ParseIP(constants.KokerBridgeDefaultIP),
		Dst:       nil,
	}

	if err := netlink.RouteAdd(&route); err != nil {
		log.Error().Err(err).Str("veth1", veth1Name).
			Msg("Unable to add route for virtual ethernet")
	}

	return nil
}

func SetupLocalInterface() {
	links, _ := netlink.LinkList()
	for _, link := range links {
		if link.Attrs().Name == "lo" {
			loAddr, _ := netlink.ParseAddr("127.0.0.1/32")
			if err := netlink.AddrAdd(link, loAddr); err != nil {
				log.Warn().Msg("Unable to configure local interface!")
			}
			netlink.LinkSetUp(link)
		}
	}
}
