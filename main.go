package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gleez/peer-watch/peerwatch"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var (

	// LDFLAGS should overwrite these variables in build time.
	Version  string
	Revision string

	labelSelector = flag.String("label-selector", "app=peer-watch", "The label to watch against pods")
	namespace     = flag.String("namespace", apiv1.NamespaceDefault, "The Kubernetes namespace for the pods")
	inCluster     = flag.Bool("use-cluster-credentials", false, "Should this request use cluster credentials?")
	kubeconfig    = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	addr          = flag.String("http", ":4040", "If non-empty, stand up a simple webserver that reports the peer state")
	debug         = flag.Bool("debug", false, "Should enable debug?")
	versionFlag   = flag.Bool("version", false, "display version and exit")

	selfUrl     string
	urlSet           = make(UrlSet)
	initialized bool = false
)

func makeClient() (*kubernetes.Clientset, error) {
	var cfg *rest.Config
	var err error

	if *inCluster {
		if cfg, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	} else {
		if *kubeconfig != "" {
			if cfg, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
				return nil, err
			}
		}
	}

	if cfg == nil {
		return nil, fmt.Errorf("peer-watch: config is not set")
	}

	return kubernetes.NewForConfig(rest.AddUserAgent(cfg, "peer-watch"))
}

func webHandler(res http.ResponseWriter, req *http.Request) {
	podUrls := urlSet.Keys()

	data, err := json.Marshal(podUrls)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write(data)
}

func validateFlags() {
	if *kubeconfig == "" && *inCluster == false {
		klog.Fatal("peer-watch: both --kubeconfig and --use-cluster-credentials cannot be empty")
	}

	if n := os.Getenv("POD_IP"); n == "" {
		klog.Fatal("peer-watch: pod ip env value cannot be empty")
	}
}

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("peer-watch: version=%s revision=%s\n", Version, Revision)
		os.Exit(0)
	}

	validateFlags()

	myIp := os.Getenv("POD_IP")

	if n := os.Getenv("POD_CACHE_NAMESPACE"); n != "" {
		*namespace = n
	}

	if ls := os.Getenv("POD_CACHE_LABEL_SELECTOR"); ls != "" {
		*labelSelector = ls
	}

	client, err := makeClient()
	if err != nil {
		klog.Fatalf("peer-watch: error connecting to the client: %v", err)
	}

	// listen for interrupts or the Linux SIGTERM signal and cancel
	// our context, which the leader election code will observe and
	// step down
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		klog.Info("peer-watch: Received termination, signaling shutdown")
		// cancel()
	}()

	opts := metav1.ListOptions{LabelSelector: *labelSelector}

	initialPeers, err := peerwatch.Init(myIp, *namespace, opts, monitorPodsFn, client, *debug)
	if err != nil {
		// Setup groupcache with just self as peer
		klog.Infof("peer-watch: WARNING error getting initial pods: %v", err)

		if *debug {
			klog.Infof("peer-watch: init %s", podUrl(myIp))
		}

		urlSet[podUrl(myIp)] = true
		initialized = true
		return
	}

	if err == nil {
		for _, ip := range initialPeers {
			if ip != "" {
				urlSet[podUrl(ip)] = true
			}
		}

		if *debug {
			klog.Infof("peer-watch: init %s", podUrl(myIp))
		}

		initialized = true
	}

	if len(*addr) > 0 {
		klog.Infof("peer-watch: http server starting at %s", *addr)
		http.HandleFunc("/", webHandler)
		klog.Fatal(http.ListenAndServe(*addr, nil))
	} else {
		select {}
	}
}

func monitorPodsFn(ip string, state peerwatch.NotifyState) {
	for !initialized {
		// prevent race condition by waiting for initial peers to be setup before processing any changes
		time.Sleep(100 * time.Millisecond)
	}

	klog.Infof("peer-watch: Got notify: %s [%d]", ip, state)

	switch state {
	case peerwatch.Added:
		urlSet[podUrl(ip)] = true
	case peerwatch.Removed:
		delete(urlSet, podUrl(ip))
	default:
		return
	}

	podUrls := urlSet.Keys()

	klog.Infof("peer-watch: New pod list = %v", podUrls)
}

func podUrl(podIp string) string {
	// return fmt.Sprintf("http://%s:%d", podIp, *port)
	return strings.TrimSpace(podIp)
}

type UrlSet map[string]bool

func (urlSet UrlSet) Keys() []string {
	keys := make([]string, len(urlSet))
	i := 0
	for key := range urlSet {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}

func (urlSet UrlSet) String() string {
	return fmt.Sprintf("%v", urlSet.Keys())
}
