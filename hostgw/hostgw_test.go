package hostgw

import (
	"net"
	"testing"

	"github.com/vishvananda/netlink"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRouteEquals(t *testing.T) {
	_, dst0, _ := net.ParseCIDR("192.168.0.0/24")
	t0 := &netlink.Route{
		Dst: dst0,
		Gw:  net.ParseIP("192.168.1.1"),
	}

	_, dst1, _ := net.ParseCIDR("192.168.0.0/24")
	t1 := &netlink.Route{
		Dst: dst1,
		Gw:  net.ParseIP("192.168.1.1"),
	}

	if !routeEquals(t0, t1) {
		t.Error("shit")
	}
}

func TestNode2Router(t *testing.T) {
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-node-0",
			Namespace: "default",
		},
		Spec: v1.NodeSpec{
			PodCIDR: "172.28.1.0/24",
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				v1.NodeAddress{
					Address: "test-node-0",
					Type:    v1.NodeHostName,
				},
				v1.NodeAddress{
					Address: "192.168.1.10",
					Type:    v1.NodeInternalIP,
				},
			},
		},
	}

	r, err := node2Router(node)
	if err != nil {
		t.Fatalf("node2Router failed: %+v", err)
	}

	t.Log(r)
}
