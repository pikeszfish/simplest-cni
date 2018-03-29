package hostgw

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/containernetworking/cni/libcni"
	log "github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"k8s.io/api/core/v1"
)

var (
	// ErrNoPodCIDR it may happen when kube-controller-manager not update pod CIDR for a new node that in time.
	ErrNoPodCIDR = errors.New("no pod CIDR found")
	// ErrNoPodAddress if a node does not have a address
	ErrNoPodAddress = errors.New("no pod address found")
	// ErrCNIConfNonexist if no .conf/.conflist/.json file found in /etc/cni/net.d/
	ErrCNIConfNonexist = errors.New("cni conf nonexist")
)

func generateRoute(node *v1.Node) (*netlink.Route, error) {
	subnetStr := node.Spec.PodCIDR
	if subnetStr == "" {
		return nil, ErrNoPodCIDR
	}

	address := ""
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			address = addr.Address
			break
		}
	}

	if address == "" {
		return nil, ErrNoPodAddress
	}

	_, subnet, _ := net.ParseCIDR(subnetStr)
	ip := net.ParseIP(address)

	// The scope of a route in Linux is an indicator of the distance to the destination network.

	// Host
	// 	A route has host scope when it leads to a destination address on the local host.
	// Link
	// 	A route has link scope when it leads to a destination address on the local network.
	// Universe
	// 	A route has universe scope when it leads to addresses more than one hop away.
	return &netlink.Route{
		Gw:    ip,
		Dst:   subnet,
		Scope: unix.RT_SCOPE_UNIVERSE,
	}, nil
}

func generateCNIConfig(r *netlink.Route, p string) error {
	files, err := libcni.ConfFiles(p, []string{".conf", ".conflist", ".json"})
	if err != nil {
		return err
	} else if len(files) == 0 {
		return ErrCNIConfNonexist
	}
	sort.Strings(files)

	filename := files[0]

	a := fmt.Sprintf(`s/__SUBNET_STR__/%s/g`, strings.Replace(r.Dst.String(), `/`, `\/`, 1))
	log.Infof("%s %s %s %s", "sed", "-i", a, filename)
	cmd := exec.Command("sed", "-i", a, filename)
	err = cmd.Run()
	if err != nil {
		log.Errorf("exec sed failed %+v", err)
		return err
	}
	return nil
}

// Handler ResourceEventHandler
type Handler struct {
	CNIBinDir  string
	CNIConfDir string
}

// OnAdd handle ADDED event
func (h *Handler) OnAdd(obj interface{}) {
	node := obj.(*v1.Node)
	log.Infof("OnAdd: <%s:%s %s>", node.Namespace, node.Name, node.Spec.PodCIDR)

	r, err := generateRoute(node)
	if err == ErrNoPodAddress || err == ErrNoPodCIDR {
		return
	} else if err != nil {
		return
	}

	hostname, _ := os.Hostname()
	if node.Name == hostname {
		err = generateCNIConfig(r, h.CNIConfDir)
		if err != nil {
			log.Errorf("generateCNIConfig err %+v", err)
			return
		}
		// self route use LINK
		r.Scope = unix.RT_SCOPE_LINK
	}
	// TODO
	// check if a route exist already.
	// allRoutes, err := netlink.RouteList()
	// if err != nil {
	// 	log.Errorf("RouteList failed")
	// 	return
	// }
	// for _, rr := range allRoutes {
	// 	if r.Equal(rr) {
	// 		log.Info("route exists!")
	// 		return
	// 	}
	// }
	if err = netlink.RouteAdd(r); err != nil {
		log.Errorf("adding route %+v to %s failed: %+v", r, node.Name, err)
	}
}

// OnDelete handle DELETED event
func (h *Handler) OnDelete(obj interface{}) {
	node := obj.(*v1.Node)
	log.Infof("OnDelete: <%s:%s %s>", node.Namespace, node.Name, node.Spec.PodCIDR)

	r, err := generateRoute(node)
	if err == ErrNoPodAddress || err == ErrNoPodCIDR {
		return
	} else if err != nil {
		return
	}

	hostname, _ := os.Hostname()
	if node.Name == hostname {
		return
	}

	log.Infof("deleting route %+v to %s", r, node.Name)
	if err = netlink.RouteDel(r); err != nil {
		log.Errorf("deleting route %+v to %s failed: %+v", r, node.Name, err)
	}
}

// OnUpdate handle `MODIFIED` event
func (h *Handler) OnUpdate(oldObj, newObj interface{}) {
	oldNode := oldObj.(*v1.Node)
	newNode := newObj.(*v1.Node)
	oldR, err := generateRoute(oldNode)
	if err == ErrNoPodAddress || err == ErrNoPodCIDR {
		return
	} else if err != nil {
		return
	}
	newR, err := generateRoute(newNode)
	if err == ErrNoPodAddress || err == ErrNoPodCIDR {
		return
	} else if err != nil {
		return
	}

	if (*oldR).Equal(*newR) {
		log.Info("Just SKIP OnUpdate!!!")
		return
	}
	h.OnDelete(oldObj)
	h.OnAdd(newObj)
}
