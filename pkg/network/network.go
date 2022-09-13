package network

import (
	"net"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
)

// SetupBridge sets up the default bridge interface.
// To keep things simple, assign ip 172.69.0.1 to it (yeah, I fixed it!)
// which is from the range of IPs which will also use for the containers
func SetupBridge(name, ip string) error {
	log.Info().Str("bridge", name).Msg("Setup default bridge")

	// Create bridge if does not exist
	log.Debug().Str("bridge", name).Msg("Create default bridge")
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = name
	bridge := &netlink.Bridge{LinkAttrs: linkAttrs}
	if err := netlink.LinkAdd(bridge); err != nil {
		return err
	}

	log.Debug().Str("bridge", name).Str("ip", ip).
		Msg("Add IP address if there is not")
	addrList, err := netlink.AddrList(bridge, 0)
	if err != nil {
		return err
	}

	if len(addrList) < 1 {
		addr, err := netlink.ParseAddr(ip)
		if err != nil {
			return err
		}
		if err := netlink.AddrAdd(bridge, addr); err != nil {
			return err
		}
	}

	// Setup the bridge
	log.Debug().Str("bridge", name).Msg("Enable default bridge")
	return netlink.LinkSetUp(bridge)
}

// CheckBridgeUp check whether the bridge is up
func CheckBridgeUp(name string) (bool, error) {
	log.Info().Str("bridge", name).
		Msg("Check default bridge is up or not")
	links, err := netlink.LinkList()
	if err == nil {
		for _, link := range links {
			if link.Type() == "bridge" && link.Attrs().Name == name {
				return true, nil
			}
		}
	}

	return false, err
}

// SetupVirtualEthernet creates and configures a virtual ethernet on host
func SetupVirtualEthernet(name, peer string) error {
	log.Info().Str("virt", name).Str("peer", peer).Msg("Setup virtual ethernet")
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = name
	vth := &netlink.Veth{
		LinkAttrs: linkAttrs,
		PeerName:  peer,
	}

	if err := netlink.LinkAdd(vth); err != nil {
		return err
	}

	return netlink.LinkSetUp(vth)
}

// LinkSetMaster sets the master of the link device
func LinkSetMaster(linkName, masterName string) error {
	log.Info().Str("link", linkName).Str("master", masterName).
		Msg("Set the master of the link device")
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return errors.Wrapf(err, "can't find link %s", linkName)
	}
	masterLink, err := netlink.LinkByName(masterName)
	if err != nil {
		return errors.Wrapf(err, "can't find link %s", linkName)
	}
	return netlink.LinkSetMaster(link, masterLink)
}

// LinkAddGateway adds route for the system
// set gateway for the link device
func LinkAddGateway(linkName, gatewayIP string) error {
	log.Info().Str("link", linkName).Str("gateway", gatewayIP).
		Msg("Set gateway for the link device")
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return err
	}
	newRoute := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Scope:     netlink.SCOPE_UNIVERSE,
		Gw:        net.ParseIP(gatewayIP),
	}

	return netlink.RouteAdd(newRoute)
}

// LinkAddAddr adds an IP address to the link device
func LinkAddAddr(linkName, ip string) error {
	log.Info().Str("link", linkName).Str("ip", ip).
		Msg("Add IP address to the ip device")
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return err
	}
	addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return errors.Wrapf(err, "can't parse %s", ip)
	}
	return netlink.AddrAdd(link, addr)
}

// LinkSetup finds and enables the link device
func LinkSetup(linkName string) error {
	log.Info().Str("link", linkName).Msg("Enable the link device")
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(link)
}

// LinkRename sets the name of the link device
func LinkRename(old, new string) error {
	log.Info().Str("oldname", old).Str("newname", new).
		Msg("Change the name of the link device")
	link, err := netlink.LinkByName(old)
	if err != nil {
		return err
	}
	return netlink.LinkSetName(link, new)
}

// IPExists checks IP is used or not
func IPExists(ip net.IP) (bool, error) {
	log.Debug().Str("ip", ip.String()).Msg("Check IP exists")
	linkList, err := netlink.AddrList(nil, 0)
	if err != nil {
		return false, err
	}
	for _, link := range linkList {
		if link.IP.Equal(ip) {
			return true, nil
		}
	}
	return false, nil
}
