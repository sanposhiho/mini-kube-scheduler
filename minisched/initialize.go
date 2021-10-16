package minisched

import (
	"fmt"

	"github.com/sanposhiho/mini-kube-scheduler/minisched/plugins/score/nodenumber"
	"github.com/sanposhiho/mini-kube-scheduler/minisched/queue"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable"
)

type Scheduler struct {
	SchedulingQueue *queue.SchedulingQueue

	client clientset.Interface

	filterPlugins   []framework.FilterPlugin
	preScorePlugins []framework.PreScorePlugin
	scorePlugins    []framework.ScorePlugin
}

// =======
// funcs for initialize
// =======

func New(
	client clientset.Interface,
	informerFactory informers.SharedInformerFactory,
) (*Scheduler, error) {
	sched := &Scheduler{
		SchedulingQueue: queue.New(),
		client:          client,
	}

	filterP, err := createFilterPlugins()
	if err != nil {
		return nil, fmt.Errorf("create filter plugins: %w", err)
	}
	sched.filterPlugins = filterP

	preScoreP, err := createPreScorePlugins()
	if err != nil {
		return nil, fmt.Errorf("create pre score plugins: %w", err)
	}
	sched.preScorePlugins = preScoreP

	scoreP, err := createScorePlugins()
	if err != nil {
		return nil, fmt.Errorf("create score plugins: %w", err)
	}
	sched.scorePlugins = scoreP

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

func createPreScorePlugins() ([]framework.PreScorePlugin, error) {
	// nodenumber is FilterPlugin.
	nodenumberplugin, err := createNodeNumberPlugin()
	if err != nil {
		return nil, fmt.Errorf("create nodenumber plugin: %w", err)
	}

	// We use nodenumber plugin only.
	preScorePlugins := []framework.PreScorePlugin{
		nodenumberplugin.(framework.PreScorePlugin),
	}

	return preScorePlugins, nil
}

func createScorePlugins() ([]framework.ScorePlugin, error) {
	// nodenumber is FilterPlugin.
	nodenumberplugin, err := createNodeNumberPlugin()
	if err != nil {
		return nil, fmt.Errorf("create nodenumber plugin: %w", err)
	}

	// We use nodenumber plugin only.
	filterPlugins := []framework.ScorePlugin{
		nodenumberplugin.(framework.ScorePlugin),
	}

	return filterPlugins, nil
}

// =====
// initialize plugins
// =====
//
// we only use nodeunschedulable and nodenumber
// Original kube-scheduler is implemented so that we can select which plugins to enable

var (
	nodeunschedulableplugin framework.Plugin
	nodenumberplugin        framework.Plugin
)

func createNodeUnschedulablePlugin() (framework.Plugin, error) {
	if nodeunschedulableplugin != nil {
		return nodeunschedulableplugin, nil
	}

	p, err := nodeunschedulable.New(nil, nil)
	nodeunschedulableplugin = p

	return p, err
}

func createNodeNumberPlugin() (framework.Plugin, error) {
	if nodenumberplugin != nil {
		return nodenumberplugin, nil
	}

	p, err := nodenumber.New(nil, nil)
	nodenumberplugin = p

	return p, err
}
