package minisched

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodename"

	"k8s.io/kubernetes/pkg/scheduler/framework"

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

	// get nodes
	nodes, err := sched.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Error(err)
		return
	}
	klog.Info("minischeduler: Get Nodes successfully")
	klog.Info("minischeduler: got nodes: ", nodes)

	// filter
	fasibleNodes, status := sched.RunFilterPlugins(ctx, nil, pod, nodes.Items)
	if !status.IsSuccess() {
		klog.Error(status.AsError())
		return
	}
	if len(fasibleNodes) == 0 {
		klog.Info("no fasible nodes for " + pod.Name)
		return
	}

	// select node randomly
	rand.Seed(time.Now().UnixNano())
	selectedNode := fasibleNodes[rand.Intn(len(nodes.Items))]

	if err := sched.Bind(ctx, nil, pod, selectedNode.Name); err != nil {
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
