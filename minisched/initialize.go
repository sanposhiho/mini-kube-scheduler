package minisched

import (
	"fmt"

	"github.com/sanposhiho/mini-kube-scheduler/minisched/plugins/score/nodenumber"
	"github.com/sanposhiho/mini-kube-scheduler/minisched/queue"
	"github.com/sanposhiho/mini-kube-scheduler/minisched/waitingpod"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable"
)

type Scheduler struct {
	SchedulingQueue *queue.SchedulingQueue

	client clientset.Interface

	waitingPods map[types.UID]*waitingpod.WaitingPod

	filterPlugins   []framework.FilterPlugin
	preScorePlugins []framework.PreScorePlugin
	scorePlugins    []framework.ScorePlugin
	permitPlugins   []framework.PermitPlugin
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
		waitingPods:     map[types.UID]*waitingpod.WaitingPod{},
	}

	filterP, err := createFilterPlugins(sched)
	if err != nil {
		return nil, fmt.Errorf("create filter plugins: %w", err)
	}
	sched.filterPlugins = filterP

	preScoreP, err := createPreScorePlugins(sched)
	if err != nil {
		return nil, fmt.Errorf("create pre score plugins: %w", err)
	}
	sched.preScorePlugins = preScoreP

	scoreP, err := createScorePlugins(sched)
	if err != nil {
		return nil, fmt.Errorf("create score plugins: %w", err)
	}
	sched.scorePlugins = scoreP

	permitP, err := createPermitPlugins(sched)
	if err != nil {
		return nil, fmt.Errorf("create permit plugins: %w", err)
	}
	sched.permitPlugins = permitP

	addAllEventHandlers(sched, informerFactory)

	return sched, nil
}

func createFilterPlugins(h waitingpod.Handle) ([]framework.FilterPlugin, error) {
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

func createPreScorePlugins(h waitingpod.Handle) ([]framework.PreScorePlugin, error) {
	// nodenumber is FilterPlugin.
	nodenumberplugin, err := createNodeNumberPlugin(h)
	if err != nil {
		return nil, fmt.Errorf("create nodenumber plugin: %w", err)
	}

	// We use nodenumber plugin only.
	preScorePlugins := []framework.PreScorePlugin{
		nodenumberplugin.(framework.PreScorePlugin),
	}

	return preScorePlugins, nil
}

func createScorePlugins(h waitingpod.Handle) ([]framework.ScorePlugin, error) {
	// nodenumber is FilterPlugin.
	nodenumberplugin, err := createNodeNumberPlugin(h)
	if err != nil {
		return nil, fmt.Errorf("create nodenumber plugin: %w", err)
	}

	// We use nodenumber plugin only.
	filterPlugins := []framework.ScorePlugin{
		nodenumberplugin.(framework.ScorePlugin),
	}

	return filterPlugins, nil
}

func createPermitPlugins(h waitingpod.Handle) ([]framework.PermitPlugin, error) {
	// nodenumber is PermitPlugin.
	nodenumberplugin, err := createNodeNumberPlugin(h)
	if err != nil {
		return nil, fmt.Errorf("create nodenumber plugin: %w", err)
	}

	// We use nodenumber plugin only.
	permitPlugins := []framework.PermitPlugin{
		nodenumberplugin.(framework.PermitPlugin),
	}

	return permitPlugins, nil
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

func createNodeNumberPlugin(h waitingpod.Handle) (framework.Plugin, error) {
	if nodenumberplugin != nil {
		return nodenumberplugin, nil
	}

	p, err := nodenumber.New(nil, h)
	nodenumberplugin = p

	return p, err
}
