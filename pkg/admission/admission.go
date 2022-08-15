package admission

import (
	"os"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

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
	StopCh              chan struct{}
	Client              kubernetes.Interface
	InformerResync      time.Duration
	ExtraSyncWait       time.Duration
	config.Config
}

func Start(ctx ControllerContext) {
	if ctx.GetCertExpiration() >= ctx.GetCertValidityBound() {
		log.WithFields(log.Fields{
			"expirationThreshold": ctx.GetCertExpiration(),
			"validityBound":       ctx.GetCertValidityBound(),
		}).Error("Validity of the certificate has to be greater than the expiration threshold")
		return
	}
	log.Info("Starting Certificate Manager Controller")
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
		ctx.IsLeaderFunc,
		ctx.LeaderSubscribeFunc(),
		secretConfig,
	)

	go secretController.Run(ctx.StopCh)

	secretInformers.Start(ctx.StopCh)
	err := controller.SyncInformers(secretInformers.Core().V1().Secrets().Informer(), ctx.ExtraSyncWait)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Could not sync Informers")
		return
	}
	<-ctx.StopCh
	log.Info("Stopping Certificate Manager Controller")
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
}
