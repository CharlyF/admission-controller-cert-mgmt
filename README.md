# Certificate Manager Library

This library can be used as a lightweight replacement of the [certificate manager](https://cert-manager.io/).

It's use is limited to self signed certificates from the APIServer.

## Example

```go
package main

import (
	"time"
	"github.com/CharlyF/admission-controller-cert-mgmt/pkg/admission"
	"github.com/CharlyF/admission-controller-cert-mgmt/pkg/config"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // to test in GKE
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	cl, err := GetKubeClient(5 * time.Second)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error getting Kubernetes client")
		return
	}

	crt := config.NewCertConfig(1*time.Hour, 2*time.Hour)
	cfg := config.NewConfig("default", "default-secret", "my-service", crt)
	ctx := admission.ControllerContext{
		IsLeaderFunc:        isLeader,
		LeaderSubscribeFunc: subscriber,
		Client:              cl,
		InformerResync:      300 * time.Second,
		Config:              cfg,
	}

	admission.Start(ctx)
}

func GetKubeClient(timeout time.Duration) (kubernetes.Interface, error) {
	clientConfig, err := getClientConfig(timeout)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(clientConfig)
}

func getClientConfig(timeout time.Duration) (*rest.Config, error) {
	var clientConfig *rest.Config
	var err error
	cfgPath := "<Insert path of kubeconfig>"
	clientConfig, err = clientcmd.BuildConfigFromFlags("", cfgPath)
	if err != nil {
		return nil, err
	}
	clientConfig.Timeout = timeout
	return clientConfig, err
}

func isLeader() bool {
	return true
}

func subscriber() <-chan struct{} {
	notification := make(chan struct{})
	go func() {
		time.Sleep(time.Second * 4) // notification after 4 seconds
		notification <- struct{}{}
	}()
	return notification
}
```