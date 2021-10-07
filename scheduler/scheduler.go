package scheduler

import (
	"context"

	"github.com/sanposhiho/mini-kube-scheduler/minisched"

	"golang.org/x/xerrors"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
	v1beta2config "k8s.io/kube-scheduler/config/v1beta2"
	"k8s.io/kubernetes/pkg/scheduler"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"
	"k8s.io/kubernetes/pkg/scheduler/apis/config/v1beta2"

	"github.com/sanposhiho/mini-kube-scheduler/scheduler/defaultconfig"
	"github.com/sanposhiho/mini-kube-scheduler/scheduler/plugin"
)

// Service manages scheduler.
type Service struct {
	// function to shutdown scheduler.
	shutdownfn func()

	clientset           clientset.Interface
	restclientCfg       *restclient.Config
	currentSchedulerCfg *v1beta2config.KubeSchedulerConfiguration
}

// NewSchedulerService starts scheduler and return *Service.
func NewSchedulerService(client clientset.Interface, restclientCfg *restclient.Config) *Service {
	return &Service{clientset: client, restclientCfg: restclientCfg}
}

func (s *Service) RestartScheduler(cfg *v1beta2config.KubeSchedulerConfiguration) error {
	s.ShutdownScheduler()

	if err := s.StartScheduler(cfg); err != nil {
		return xerrors.Errorf("start scheduler: %w", err)
	}
	return nil
}

// StartScheduler starts scheduler.
func (s *Service) StartScheduler(versionedcfg *v1beta2config.KubeSchedulerConfiguration) error {
	clientSet := s.clientset
	ctx, cancel := context.WithCancel(context.Background())

	informerFactory := scheduler.NewInformerFactory(clientSet, 0)
	evtBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{
		Interface: clientSet.EventsV1(),
	})

	evtBroadcaster.StartRecordingToSink(ctx.Done())

	s.currentSchedulerCfg = versionedcfg.DeepCopy()

	sched := minisched.New(
		clientSet,
		informerFactory,
	)

	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())

	go sched.Run(ctx)

	s.shutdownfn = cancel

	return nil
}

func (s *Service) ShutdownScheduler() {
	if s.shutdownfn != nil {
		klog.Info("shutdown scheduler...")
		s.shutdownfn()
	}
}

func (s *Service) GetSchedulerConfig() *v1beta2config.KubeSchedulerConfiguration {
	return s.currentSchedulerCfg
}

// convertConfigurationForSimulator convert KubeSchedulerConfiguration to apply scheduler on simulator
// (1) It excludes non-allowed changes. Now, we accept only changes to Profiles.Plugins field.
// (2) It replaces filter/score default-plugins with plugins for simulator.
// (3) It convert KubeSchedulerConfiguration from v1beta2config.KubeSchedulerConfiguration to config.KubeSchedulerConfiguration.
func convertConfigurationForSimulator(versioned *v1beta2config.KubeSchedulerConfiguration) (*config.KubeSchedulerConfiguration, error) {
	if len(versioned.Profiles) == 0 {
		defaultSchedulerName := v1.DefaultSchedulerName
		versioned.Profiles = []v1beta2config.KubeSchedulerProfile{
			{
				SchedulerName: &defaultSchedulerName,
				Plugins:       &v1beta2config.Plugins{},
			},
		}
	}

	for i := range versioned.Profiles {
		if versioned.Profiles[i].Plugins == nil {
			versioned.Profiles[i].Plugins = &v1beta2config.Plugins{}
		}

		plugins, err := plugin.ConvertForSimulator(versioned.Profiles[i].Plugins)
		if err != nil {
			return nil, xerrors.Errorf("convert plugins for simulator: %w", err)
		}
		versioned.Profiles[i].Plugins = plugins

		pluginConfigForSimulatorPlugins, err := plugin.NewPluginConfig(versioned.Profiles[i].PluginConfig)
		if err != nil {
			return nil, xerrors.Errorf("get plugin configs: %w", err)
		}
		versioned.Profiles[i].PluginConfig = pluginConfigForSimulatorPlugins
	}

	defaultCfg, err := defaultconfig.DefaultSchedulerConfig()
	if err != nil {
		return nil, xerrors.Errorf("get default scheduler config: %w", err)
	}

	// set default value to all field other than Profiles.
	defaultCfg.Profiles = versioned.Profiles
	versioned = defaultCfg

	v1beta2.SetDefaults_KubeSchedulerConfiguration(versioned)
	cfg := config.KubeSchedulerConfiguration{}
	if err := scheme.Scheme.Convert(versioned, &cfg, nil); err != nil {
		return nil, xerrors.Errorf("convert configuration: %w", err)
	}

	return &cfg, nil
}
