package minisched

import (
	"context"
	"math/rand"
	"time"

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
}

func New(
	client clientset.Interface,
	informerFactory informers.SharedInformerFactory,
) *Scheduler {
	sched := &Scheduler{
		SchedulingQueue: queue.New(),
		client:          client,
	}

	addAllEventHandlers(sched, informerFactory)

	return sched
}

func (sched *Scheduler) Run(ctx context.Context) {
	wait.UntilWithContext(ctx, sched.scheduleOne, 0)
}

func (sched *Scheduler) scheduleOne(ctx context.Context) {
	klog.Info("minischeduler: Try to get pod from queue....")
	pod := sched.SchedulingQueue.NextPod()
	klog.Info("minischeduler: Start schedule(" + pod.Name + ")")

	// get nodes
	nodes, err := sched.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Error(err)
		return
	}
	klog.Info("minischeduler: Got Nodes successfully")

	// select node randomly
	rand.Seed(time.Now().UnixNano())
	selectedNode := nodes.Items[rand.Intn(len(nodes.Items))]

	if err := sched.Bind(ctx, nil, pod, selectedNode.Name); err != nil {
		klog.Error(err)
		return
	}

	klog.Info("minischeduler: Bind Pod successfully")
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
