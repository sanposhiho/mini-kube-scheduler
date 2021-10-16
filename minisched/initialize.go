package minisched

import (
	"fmt"

	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable"

	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/sanposhiho/mini-kube-scheduler/minisched/queue"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
)

type Scheduler struct {
	SchedulingQueue *queue.SchedulingQueue

	client clientset.Interface

	filterPlugins []framework.FilterPlugin
}

// =======
// funcs for initialize
// =======

func New(
	client clientset.Interface,
	informerFactory informers.SharedInformerFactory,
) (*Scheduler, error) {
	filterP, err := createFilterPlugins()
	if err != nil {
		return nil, fmt.Errorf("create filter plugins: %w", err)
	}

	sched := &Scheduler{
		SchedulingQueue: queue.New(),
		client:          client,

		filterPlugins: filterP,
	}

	addAllEventHandlers(sched, informerFactory)

	return sched, nil
}

func createFilterPlugins() ([]framework.FilterPlugin, error) {
	// nodeunschedulable is FilterPlugin.
	nodeunschedulableplugin, err := createNodeUnschedulablePlugin()
	if err != nil {
		return nil, fmt.Errorf("create nodeunschedulable plugin: %w", err)
	}

	// We use nodeunschedulable plugin only.
	filterPlugins := []framework.FilterPlugin{
		nodeunschedulableplugin.(framework.FilterPlugin),
	}

	return filterPlugins, nil
}

// =====
// initialize plugins
// =====
//
// we only use nodeunschedulable

var (
	nodeunschedulableplugin framework.Plugin
)

func createNodeUnschedulablePlugin() (framework.Plugin, error) {
	if nodeunschedulableplugin != nil {
		return nodeunschedulableplugin, nil
	}

	p, err := nodeunschedulable.New(nil, nil)
	nodeunschedulableplugin = p

	return p, err
}
