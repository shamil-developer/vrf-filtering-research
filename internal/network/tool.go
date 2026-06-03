package network

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type Tool struct{}

func NewTool() *Tool {
	return &Tool{}
}

func (t *Tool) InNamespace(
	name string,
	fn func() error,
) error {
	if name == "" || name == "root" {
		return fn()
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	current, err := netns.Get()
	if err != nil {
		return fmt.Errorf("get current namespace: %w", err)
	}
	defer current.Close()

	target, err := netns.GetFromName(name)
	if err != nil {
		return fmt.Errorf("open namespace %q: %w", name, err)
	}
	defer target.Close()

	if err := netns.Set(target); err != nil {
		return fmt.Errorf("set namespace %q: %w", name, err)
	}

	defer func() {
		_ = netns.Set(current)
	}()

	return fn()
}

func (t *Tool) CreateNamespace(
	name string,
) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	current, err := netns.Get()
	if err != nil {
		return fmt.Errorf("get current namespace: %w", err)
	}
	defer current.Close()

	ns, err := netns.NewNamed(name)
	if err != nil {
		return fmt.Errorf("create namespace %q: %w", name, err)
	}
	defer ns.Close()

	if err := netns.Set(current); err != nil {
		return fmt.Errorf("restore namespace after creating %q: %w", name, err)
	}

	return nil
}

func (t *Tool) CreateBridge(
	namespace string,
	name string,
) error {
	return t.InNamespace(namespace, func() error {
		link := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: name,
			},
		}

		if err := netlink.LinkAdd(link); err != nil {
			return fmt.Errorf("create bridge %q: %w", name, err)
		}

		return nil
	})
}

func (t *Tool) CreateVRF(
	namespace string,
	name string,
	table uint32,
) error {
	return t.InNamespace(namespace, func() error {
		link := &netlink.Vrf{
			LinkAttrs: netlink.LinkAttrs{
				Name: name,
			},
			Table: table,
		}

		if err := netlink.LinkAdd(link); err != nil {
			return fmt.Errorf("create vrf %q: %w", name, err)
		}

		return nil
	})
}

func (t *Tool) CreateVeth(
	namespace string,
	left string,
	right string,
) error {
	return t.InNamespace(namespace, func() error {
		link := &netlink.Veth{
			LinkAttrs: netlink.LinkAttrs{
				Name: left,
			},
			PeerName: right,
		}

		if err := netlink.LinkAdd(link); err != nil {
			return fmt.Errorf("create veth %q/%q: %w", left, right, err)
		}

		return nil
	})
}

func (t *Tool) MoveInterface(
	namespace string,
	iface string,
	targetNamespace string,
) error {
	target, err := netns.GetFromName(targetNamespace)
	if err != nil {
		return fmt.Errorf("open target namespace %q: %w", targetNamespace, err)
	}
	defer target.Close()

	return t.InNamespace(namespace, func() error {
		link, err := netlink.LinkByName(iface)
		if err != nil {
			return fmt.Errorf("find interface %q: %w", iface, err)
		}

		if err := netlink.LinkSetNsFd(link, int(target)); err != nil {
			return fmt.Errorf("move interface %q to %q: %w", iface, targetNamespace, err)
		}

		return nil
	})
}

func (t *Tool) AttachInterface(
	namespace string,
	parent string,
	child string,
) error {
	return t.InNamespace(namespace, func() error {
		parentLink, err := netlink.LinkByName(parent)
		if err != nil {
			return fmt.Errorf("find parent interface %q: %w", parent, err)
		}

		childLink, err := netlink.LinkByName(child)
		if err != nil {
			return fmt.Errorf("find child interface %q: %w", child, err)
		}

		if err := netlink.LinkSetMaster(childLink, parentLink); err != nil {
			return fmt.Errorf("attach interface %q to %q: %w", child, parent, err)
		}

		return nil
	})
}

func (t *Tool) AssignIP(
	namespace string,
	iface string,
	address string,
) error {
	return t.InNamespace(namespace, func() error {
		link, err := netlink.LinkByName(iface)
		if err != nil {
			return fmt.Errorf("find interface %q: %w", iface, err)
		}

		addr, err := netlink.ParseAddr(address)
		if err != nil {
			return fmt.Errorf("parse address %q: %w", address, err)
		}

		if err := netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("assign address %q to %q: %w", address, iface, err)
		}

		return nil
	})
}

func (t *Tool) InterfaceUp(
	namespace string,
	iface string,
) error {
	return t.InNamespace(namespace, func() error {
		link, err := netlink.LinkByName(iface)
		if err != nil {
			return fmt.Errorf("find interface %q: %w", iface, err)
		}

		if err := netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("set interface %q up: %w", iface, err)
		}

		return nil
	})
}

func (t *Tool) AddRoute(
	namespace string,
	dst string,
	gateway string,
	iface string,
	table int,
) error {
	return t.InNamespace(namespace, func() error {
		var route netlink.Route

		if dst != "" && dst != "default" {
			_, ipNet, err := net.ParseCIDR(dst)
			if err != nil {
				return fmt.Errorf("parse route dst %q: %w", dst, err)
			}
			route.Dst = ipNet
		}

		if gateway != "" {
			route.Gw = net.ParseIP(gateway)
			if route.Gw == nil {
				return fmt.Errorf("parse route gateway %q", gateway)
			}
		}

		if iface != "" {
			link, err := netlink.LinkByName(iface)
			if err != nil {
				return fmt.Errorf("find route interface %q: %w", iface, err)
			}
			route.LinkIndex = link.Attrs().Index
		}

		if table != 0 {
			route.Table = table
		}

		if err := netlink.RouteAdd(&route); err != nil {
			return fmt.Errorf("add route dst=%q gateway=%q iface=%q table=%d: %w", dst, gateway, iface, table, err)
		}

		return nil
	})
}

func (t *Tool) SetSysctl(
	name string,
	value string,
) error {
	path := "/proc/sys/" + strings.ReplaceAll(name, ".", "/")

	if err := os.WriteFile(path, []byte(value), 0644); err != nil {
		return fmt.Errorf("set sysctl %q=%q: %w", name, value, err)
	}

	return nil
}
