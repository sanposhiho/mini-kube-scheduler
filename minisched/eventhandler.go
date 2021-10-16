package minisched

import (
	"fmt"

	"k8s.io/kubernetes/pkg/scheduler/framework"

	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func addAllEventHandlers(
	sched *Scheduler,
	informerFactory informers.SharedInformerFactory,
	gvkMap map[framework.GVK]framework.ActionType,
) {
	// unscheduled pod
	informerFactory.Core().V1().Pods().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					return !assignedPod(t)
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				// Consider only adding.
				AddFunc: sched.addPodToSchedulingQueue,
			},
		},
	)

	buildEvtResHandler := func(at framework.ActionType, gvk framework.GVK, shortGVK string) cache.ResourceEventHandlerFuncs {
		funcs := cache.ResourceEventHandlerFuncs{}
		if at&framework.Add != 0 {
			evt := framework.ClusterEvent{Resource: gvk, ActionType: framework.Add, Label: fmt.Sprintf("%vAdd", shortGVK)}
			funcs.AddFunc = func(_ interface{}) {
				sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(evt)
			}
		}
		if at&framework.Update != 0 {
			evt := framework.ClusterEvent{Resource: gvk, ActionType: framework.Update, Label: fmt.Sprintf("%vUpdate", shortGVK)}
			funcs.UpdateFunc = func(_, _ interface{}) {
				sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(evt)
			}
		}
		if at&framework.Delete != 0 {
			evt := framework.ClusterEvent{Resource: gvk, ActionType: framework.Delete, Label: fmt.Sprintf("%vDelete", shortGVK)}
			funcs.DeleteFunc = func(_ interface{}) {
				sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(evt)
			}
		}
		return funcs
	}

	for gvk, at := range gvkMap {
		switch gvk {
		case framework.Node:
			informerFactory.Core().V1().Nodes().Informer().AddEventHandler(
				buildEvtResHandler(at, framework.Node, "Node"),
			)
			//case framework.CSINode:
			//case framework.CSIDriver:
			//case framework.CSIStorageCapacity:
			//case framework.PersistentVolume:
			//case framework.PersistentVolumeClaim:
			//case framework.StorageClass:
			//case framework.Service:
			//default:
		}

	}
}

// assignedPod selects pods that are assigned (scheduled and running).
func assignedPod(pod *v1.Pod) bool {
	return len(pod.Spec.NodeName) != 0
}

func (sched *Scheduler) addPodToSchedulingQueue(obj interface{}) {
	pod := obj.(*v1.Pod)

	if err := sched.SchedulingQueue.Add(pod); err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to queue %T: %v", obj, err))
	}
}
