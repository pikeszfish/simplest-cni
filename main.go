package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	log "github.com/golang/glog"
	"github.com/pikeszfish/simplest-cni/hostgw"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var buildstamp = "please build with -ldflags \"-X main.buildstamp=`date '+%Y-%m-%d_%I:%M:%S'`\""

func main() {
	fmt.Println("My buildstamp is:", buildstamp)

	resync := flag.Int("resync", 30, "Resync period in seconds")
	incluster := flag.Bool("in-cluster", true, "If this in run inside a pod")
	profile := flag.Bool("profile", false, "Enable profiling")
	address := flag.String("profile_host", "localhost", "Profiling server host")
	port := flag.Int("profile-port", 9801, "Profiling server port")
	test := flag.Bool("test", false, "Dry-run. To test if the binary is complete")
	master := flag.String("master", "", "master url")
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	cniBinDir := flag.String("cni-bin-dir", "/opt/cni/bin", "The full path of the directory in which to search for CNI plugin binaries.")
	cniConfDir := flag.String("cni-conf-dir", "/etc/cni/net.d", "The full path of the directory in which to search for CNI config files")
	flag.Parse()

	defer log.Flush()

	if *profile {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

			server := &http.Server{
				Addr:    net.JoinHostPort(*address, strconv.Itoa(*port)),
				Handler: mux,
			}
			log.Fatalf("%+v", server.ListenAndServe())
		}()
	}

	var (
		clientConfig *rest.Config
		err          error
	)

	if *incluster {
		clientConfig, err = rest.InClusterConfig()
	} else {
		clientConfig, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	}

	if err != nil {
		log.Fatalf("Unable to create config: %+v", err)
	}

	// Create kubeclient
	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		log.Fatalf("Invalid api configuration: %+v", err)
	}

	if *test {
		nodes, err := clientset.Core().Nodes().List(meta_v1.ListOptions{})
		if err != nil {
			log.Fatalf("can not connect k8s master: %v\n", err)
		}
		for i, node := range nodes.Items {
			log.Infof("%d: %v\n", i, node)
		}
		return
	}

	// create the node watcher
	listwatcher := cache.NewListWatchFromClient(clientset.Core().RESTClient(), "nodes", v1.NamespaceAll, fields.Everything())

	_, informer := cache.NewInformer(listwatcher, &v1.Node{}, time.Second*(time.Duration)(*resync), &hostgw.Handler{
		CNIBinDir:  *cniBinDir,
		CNIConfDir: *cniConfDir,
	})

	informer.Run(wait.NeverStop)
}
