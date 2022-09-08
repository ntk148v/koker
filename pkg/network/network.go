package network

import (
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/ntk148v/koker/pkg/constants"
)

// CheckBridgeUp check whether the bridge is up
func CheckBridgeUp() (bool, error) {
	log.Info().Msgf("Check default bridge `%s` is up or not", constants.KokerBridgeName)
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
	log.Info().Msgf("Setup default bridge `%s`", constants.KokerBridgeName)
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = constants.KokerBridgeName
	log.Debug().Msgf("Add a new link device `%s`", constants.KokerBridgeName)
	bridge := &netlink.Bridge{LinkAttrs: linkAttrs}
	if err := netlink.LinkAdd(bridge); err != nil {
		return err
	}

	addr, _ := netlink.ParseAddr(constants.KokerBridgeDefaultIP)
	log.Debug().Msgf("Add an IP address to bridge `%s`", constants.KokerBridgeName)
	netlink.AddrAdd(bridge, addr)
	log.Debug().Msgf("Bring bridge `%s` up", constants.KokerBridgeName)
	netlink.LinkSetUp(bridge)
	return nil
}
