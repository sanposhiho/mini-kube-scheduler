package minisched

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/sanposhiho/mini-kube-scheduler/minisched/plugins/score/nodenumber"

	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodename"

	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sanposhiho/mini-kube-scheduler/minisched/queue"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
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
	filterP, err := createFilterPlugins()
	if err != nil {
		return nil, fmt.Errorf("create filter plugins: %w", err)
	}

	preScoreP, err := createPreScorePlugins()
	if err != nil {
		return nil, fmt.Errorf("create pre score plugins: %w", err)
	}

	scoreP, err := createScorePlugins()
	if err != nil {
		return nil, fmt.Errorf("create score plugins: %w", err)
	}

	sched := &Scheduler{
		SchedulingQueue: queue.New(),
		client:          client,

		filterPlugins:   filterP,
		preScorePlugins: preScoreP,
		scorePlugins:    scoreP,
	}

	addAllEventHandlers(sched, informerFactory)

	return sched, nil
}

func createFilterPlugins() ([]framework.FilterPlugin, error) {
	// nodename is FilterPlugin.
	nodenameplugin, err := nodename.New(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create nodename plugin: %w", err)
	}

	// We use nodename plugin only.
	filterPlugins := []framework.FilterPlugin{
		nodenameplugin.(framework.FilterPlugin),
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

var nodenumberplugin framework.Plugin

func createNodeNumberPlugin() (framework.Plugin, error) {
	if nodenumberplugin != nil {
		return nodenumberplugin, nil
	}

	p, err := nodenumber.New(nil, nil)
	nodenumberplugin = p

	return p, err
}

// ======
// main logic
// ======

func (sched *Scheduler) Run(ctx context.Context) {
	wait.UntilWithContext(ctx, sched.scheduleOne, 0)
}

func (sched *Scheduler) scheduleOne(ctx context.Context) {
	klog.Info("minischeduler: Try to get pod from queue....")
	pod := sched.SchedulingQueue.NextPod()
	klog.Info("minischeduler: Start schedule: pod name:" + pod.Name)

	state := framework.NewCycleState()

	// get nodes
	nodes, err := sched.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Error(err)
		return
	}
	klog.Info("minischeduler: Get Nodes successfully")
	klog.Info("minischeduler: got nodes: ", nodes)

	// filter
	fasibleNodes, status := sched.RunFilterPlugins(ctx, state, pod, nodes.Items)
	if !status.IsSuccess() {
		klog.Error(status.AsError())
		return
	}
	if len(fasibleNodes) == 0 {
		klog.Info("no fasible nodes for " + pod.Name)
		return
	}

	klog.Info("minischeduler: ran filter plugins successfully")
	klog.Info("minischeduler: fasible nodes: ", fasibleNodes)

	// pre score
	status = sched.RunPreScorePlugins(ctx, state, pod, fasibleNodes)
	if !status.IsSuccess() {
		klog.Error(status.AsError())
		return
	}
	klog.Info("minischeduler: ran pre score plugins successfully")

	// score
	score, status := sched.RunScorePlugins(ctx, state, pod, fasibleNodes)
	if !status.IsSuccess() {
		klog.Error(status.AsError())
		return
	}

	klog.Info("minischeduler: ran score plugins successfully")
	klog.Info("minischeduler: score results", score)

	hostname, err := sched.selectHost(score)
	if err != nil {
		klog.Error(err)
		return
	}

	if err := sched.Bind(ctx, nil, pod, hostname); err != nil {
		klog.Error(err)
		return
	}

	klog.Info("minischeduler: Bind Pod successfully")
}

func (sched *Scheduler) RunFilterPlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []v1.Node) ([]*v1.Node, *framework.Status) {
	feasibleNodes := make([]*v1.Node, 0, len(nodes))

	// TODO: consider about nominated pod
	for _, n := range nodes {
		n := n
		nodeInfo := framework.NewNodeInfo()
		nodeInfo.SetNode(&n)

		status := framework.NewStatus(framework.Success)
		for _, pl := range sched.filterPlugins {
			status = pl.Filter(ctx, state, pod, nodeInfo)
			if !status.IsSuccess() {
				status.SetFailedPlugin(pl.Name())
				break
			}
		}
		if status.IsSuccess() {
			feasibleNodes = append(feasibleNodes, nodeInfo.Node())
		}
	}

	return feasibleNodes, nil
}

func (sched *Scheduler) RunPreScorePlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	for _, pl := range sched.preScorePlugins {
		status := pl.PreScore(ctx, state, pod, nodes)
		if !status.IsSuccess() {
			return status
		}
	}

	return nil
}

func (sched *Scheduler) RunScorePlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) (framework.NodeScoreList, *framework.Status) {
	scoresMap := sched.createPluginToNodeScores(nodes)

	for index, n := range nodes {
		for _, pl := range sched.scorePlugins {
			score, status := pl.Score(ctx, state, pod, n.Name)
			if !status.IsSuccess() {
				return nil, status
			}
			scoresMap[pl.Name()][index] = framework.NodeScore{
				Name:  n.Name,
				Score: score,
			}
		}
	}

	// TODO: plugin weight & normalizeScore

	result := make(framework.NodeScoreList, 0, len(nodes))
	for i := range nodes {
		result = append(result, framework.NodeScore{Name: nodes[i].Name, Score: 0})
		for j := range scoresMap {
			result[i].Score += scoresMap[j][i].Score
		}
	}

	return result, nil
}

func (sched *Scheduler) Bind(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) error {
	binding := &v1.Binding{
		ObjectMeta: metav1.ObjectMeta{Namespace: p.Namespace, Name: p.Name, UID: p.UID},
		Target:     v1.ObjectReference{Kind: "Node", Name: nodeName},
	}

	err := sched.client.CoreV1().Pods(binding.Namespace).Bind(ctx, binding, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// ============
// util funcs
// ============

func (sched *Scheduler) selectHost(nodeScoreList framework.NodeScoreList) (string, error) {
	if len(nodeScoreList) == 0 {
		return "", fmt.Errorf("empty priorityList")
	}
	maxScore := nodeScoreList[0].Score
	selected := nodeScoreList[0].Name
	cntOfMaxScore := 1
	for _, ns := range nodeScoreList[1:] {
		if ns.Score > maxScore {
			maxScore = ns.Score
			selected = ns.Name
			cntOfMaxScore = 1
		} else if ns.Score == maxScore {
			cntOfMaxScore++
			if rand.Intn(cntOfMaxScore) == 0 {
				// Replace the candidate with probability of 1/cntOfMaxScore
				selected = ns.Name
			}
		}
	}
	return selected, nil
}

func (sched *Scheduler) createPluginToNodeScores(nodes []*v1.Node) framework.PluginToNodeScores {
	pluginToNodeScores := make(framework.PluginToNodeScores, len(sched.scorePlugins))
	for _, pl := range sched.scorePlugins {
		pluginToNodeScores[pl.Name()] = make(framework.NodeScoreList, len(nodes))
	}

	return pluginToNodeScores
}
