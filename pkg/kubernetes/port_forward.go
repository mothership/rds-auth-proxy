package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/mothership/rds-auth-proxy/pkg/file"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardCommand struct {
	Namespace     string
	PodName       string
	Config        *restclient.Config
	Client        restclient.Interface
	PodClient     corev1client.PodsGetter
	Ports         []string
	Address       []string
	Out, ErrOut   *bytes.Buffer
	PortForwarder *portforward.PortForwarder
	StopChannel   chan struct{}
	ReadyChannel  chan struct{}
}

type PortForwardOptions struct {
	Namespace  string
	Deployment string
	Ports      []string
	Context    string
}

func loadConfig(path, context string) (*restclient.Config, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: path}
	overrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		overrides.CurrentContext = context
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
}

func BuildPortForwardCommand(ctx context.Context, kubeConfigPath string, opts PortForwardOptions) (*PortForwardCommand, error) {
	cmd := &PortForwardCommand{
		Namespace:    opts.Namespace,
		Ports:        opts.Ports,
		Address:      []string{"localhost"},
		Out:          new(bytes.Buffer),
		ErrOut:       new(bytes.Buffer),
		StopChannel:  make(chan struct{}, 1),
		ReadyChannel: make(chan struct{}),
	}
	path, err := file.ExpandPath(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	config, err := loadConfig(path, opts.Context)
	if err != nil {
		return nil, err
	}
	cmd.Config = config
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cmd.Client = clientset.CoreV1().RESTClient()
	cmd.PodClient = clientset.CoreV1()
	pods, err := cmd.PodClient.Pods(opts.Namespace).List(ctx, v1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", opts.Deployment),
		FieldSelector: "status.phase=Running",
		Limit:         3,
	})
	if err != nil {
		return nil, err
	}

	rand.Seed(time.Now().Unix())
	cmd.PodName = pods.Items[rand.Intn(len(pods.Items))].Name
	return cmd, nil
}

// ForwardPort forwards a port until context is canceled
func ForwardPort(ctx context.Context, cmd *PortForwardCommand) error {
	go func() {
		<-ctx.Done()
		if cmd.StopChannel != nil {
			close(cmd.StopChannel)
		}
	}()

	req := cmd.Client.Post().
		Resource("pods").
		Namespace(cmd.Namespace).
		Name(cmd.PodName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(cmd.Config)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	fw, err := portforward.NewOnAddresses(dialer, cmd.Address, cmd.Ports, cmd.StopChannel, cmd.ReadyChannel, cmd.Out, cmd.ErrOut)
	if err != nil {
		return err
	}
	cmd.PortForwarder = fw
	return fw.ForwardPorts()
}
