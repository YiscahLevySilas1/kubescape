package hostsensorutils

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kubescape/k8s-interface/k8sinterface"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func NewPortForwarder(namespace, podName string, port int, stopCh chan struct{}) (*portforward.PortForwarder, error) {
	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})
	// managing termination signal from the terminal. As you can see the stopCh
	// gets closed to gracefully handle its termination.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("Bye...")
		close(stopCh)
	}()

	config := k8sinterface.GetK8sConfig()
	if config == nil {
		return nil, errors.New("failed to create new PortForwarder, config is nil")
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	hostIP := strings.TrimLeft(config.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, err
	}
	writer := io.Writer(ioutil.Discard)
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", 0, port)}, stopCh, readyCh, writer, writer)
	if err != nil {
		return nil, err
	}
	return fw, nil
}
