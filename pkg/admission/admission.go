package admission

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/CharlyF/admission-controller-cert-mgmt/pkg/config"
	"github.com/CharlyF/admission-controller-cert-mgmt/pkg/controller"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// TODO: what the lib gets passed in.
type ControllerContext struct {
	IsLeaderFunc        func() bool
	LeaderSubscribeFunc func() <-chan struct{}
	Stop                context.Context
	Client              kubernetes.Interface
	LocalPath           string
	InformerResync      time.Duration
	ExtraSyncWait       time.Duration
	config.Config
}

func Start(ctx ControllerContext) error {
	if ctx.GetCertExpiration() >= ctx.GetCertValidityBound() {
		err := fmt.Errorf("validity of the certificate: %v has to be greater than the expiration threshold: %v", ctx.GetCertExpiration(), ctx.GetCertValidityBound())
		return err
	}
	log.Info("Starting Certificate Manager Controller")
	stopCh := make(chan struct{})
	defer close(stopCh)
	nameFieldkey := "metadata.name"
	optionsForService := func(options *metav1.ListOptions) {
		options.FieldSelector = fields.OneTermEqualSelector(nameFieldkey, ctx.GetName()).String()
	}

	secretInformers := informers.NewSharedInformerFactoryWithOptions(
		ctx.Client,
		ctx.InformerResync*time.Second,
		informers.WithTweakListOptions(optionsForService),
		informers.WithNamespace(ctx.GetNs()),
	)

	certConfig := config.NewCertConfig(
		ctx.GetCertExpiration(),
		ctx.GetCertValidityBound())

	secretConfig := config.NewConfig(
		ctx.GetNs(),
		ctx.GetName(),
		ctx.GetSvc(),
		certConfig)
	secretController := controller.NewController(
		ctx.Client,
		secretInformers.Core().V1().Secrets(),
		ctx.LocalPath,
		ctx.IsLeaderFunc,
		ctx.LeaderSubscribeFunc(),
		secretConfig,
	)

	go secretController.Run(stopCh)

	secretInformers.Start(stopCh)
	err := controller.SyncInformers(secretInformers.Core().V1().Secrets().Informer(), ctx.ExtraSyncWait)
	if err != nil {
		log.Errorf("Could not sync Informers", "error", err)
		return err
	}
	<-ctx.Stop.Done()
	log.Info("Stopping Certificate Manager Controller")
	return nil
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
}
