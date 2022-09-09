package network

import (
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

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

	addr, _ := netlink.ParseAddr(constants.KokerBridgeDefaultIP)
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
