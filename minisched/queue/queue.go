package queue

import (
	"sync"
	"time"

	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/kubernetes/pkg/scheduler/framework"

	v1 "k8s.io/api/core/v1"
)

type SchedulingQueue struct {
	lock sync.RWMutex

	activeQ        []*framework.QueuedPodInfo
	podBackoffQ    []*framework.QueuedPodInfo
	unschedulableQ map[string]*framework.QueuedPodInfo

	clusterEventMap map[framework.ClusterEvent]sets.String
}

func New(clusterEventMap map[framework.ClusterEvent]sets.String) *SchedulingQueue {
	return &SchedulingQueue{
		activeQ:         []*framework.QueuedPodInfo{},
		podBackoffQ:     []*framework.QueuedPodInfo{},
		unschedulableQ:  map[string]*framework.QueuedPodInfo{},
		clusterEventMap: clusterEventMap,
	}
}

func (s *SchedulingQueue) Add(pod *v1.Pod) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	podInfo := s.newQueuedPodInfo(pod)

	s.activeQ = append(s.activeQ, podInfo)
	return nil
}

// PreEnqueueCheck is a function type. It's used to build functions that
// run against a Pod and the caller can choose to enqueue or skip the Pod
// by the checking result.
type PreEnqueueCheck func(pod *v1.Pod) bool

// MoveAllToActiveOrBackoffQueue moves all pods from unschedulableQ to activeQ or backoffQ.
// This function adds all pods and then signals the condition variable to ensure that
// if Pop() is waiting for an item, it receives the signal after all the pods are in the
// queue and the head is the highest priority pod.
func (s *SchedulingQueue) MoveAllToActiveOrBackoffQueue(event framework.ClusterEvent) {
	s.lock.Lock()
	defer s.lock.Unlock()
	unschedulablePods := make([]*framework.QueuedPodInfo, 0, len(s.unschedulableQ))
	for _, pInfo := range s.unschedulableQ {
		unschedulablePods = append(unschedulablePods, pInfo)
	}
	s.movePodsToActiveOrBackoffQueue(unschedulablePods, event)
}

// NOTE: this function assumes lock has been acquired in caller
func (s *SchedulingQueue) movePodsToActiveOrBackoffQueue(podInfoList []*framework.QueuedPodInfo, event framework.ClusterEvent) {
	for _, pInfo := range podInfoList {
		// If the event doesn't help making the Pod schedulable, continue.
		// Note: we don't run the check if pInfo.UnschedulablePlugins is nil, which denotes
		// either there is some abnormal error, or scheduling the pod failed by plugins other than PreFilter, Filter and Permit.
		// In that case, it's desired to move it anyways.
		if len(pInfo.UnschedulablePlugins) != 0 && !s.podMatchesEvent(pInfo, event) {
			continue
		}

		if isPodBackingoff(pInfo) {
			s.podBackoffQ = append(s.podBackoffQ, pInfo)
		} else {
			s.activeQ = append(s.activeQ, pInfo)
		}
		delete(s.unschedulableQ, keyFunc(pInfo))
	}
}

func (s *SchedulingQueue) NextPod() *v1.Pod {
	// wait
	for len(s.activeQ) == 0 {
	}

	p := s.activeQ[0]
	s.activeQ = s.activeQ[1:]
	return p.Pod
}

// this function is the similar to AddUnschedulableIfNotPresent on original kube-scheduler.
func (s *SchedulingQueue) AddUnschedulable(pInfo *framework.QueuedPodInfo) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Refresh the timestamp since the pod is re-added.
	pInfo.Timestamp = time.Now()

	// add or update
	s.unschedulableQ[keyFunc(pInfo)] = pInfo

	klog.Info("queue: pod added to unschedulableQ: "+pInfo.Pod.Name+". This pod is unscheduled by ", pInfo.UnschedulablePlugins)
	return nil
}

func (s *SchedulingQueue) Update(oldPod, newPod *v1.Pod) error {
	// TODO: implement
	panic("not implemented")
	return nil
}

func (s *SchedulingQueue) Delete(pod *v1.Pod) error {
	// TODO: implement
	panic("not implemented")
	return nil
}

// AssignedPodAdded is called when a bound pod is added. Creation of this pod
// may make pending pods with matching affinity terms schedulable.
func (s *SchedulingQueue) AssignedPodAdded(pod *v1.Pod) {
	// TODO: implement
	panic("not implemented")
}

// AssignedPodUpdated is called when a bound pod is updated. Change of labels
// may make pending pods with matching affinity terms schedulable.
func (s *SchedulingQueue) AssignedPodUpdated(pod *v1.Pod) {
	// TODO: implement
	panic("not implemented")
}

// flushBackoffQCompleted Moves all pods from backoffQ which have completed backoff in to activeQ
func (s *SchedulingQueue) flushBackoffQCompleted() {
	// TODO: implement
	panic("note implemented")
}

// flushUnschedulableQLeftover moves pods which stay in unschedulableQ longer than unschedulableQTimeInterval
// to backoffQ or activeQ.
func (s *SchedulingQueue) flushUnschedulableQLeftover() {
	// TODO: implement
	panic("note implemented")
}

// =====
// utils
// =====

func keyFunc(pInfo *framework.QueuedPodInfo) string {
	return pInfo.Pod.Name + "_" + pInfo.Pod.Namespace
}

func (s *SchedulingQueue) newQueuedPodInfo(pod *v1.Pod, unschedulableplugins ...string) *framework.QueuedPodInfo {
	now := time.Now()
	return &framework.QueuedPodInfo{
		PodInfo:                 framework.NewPodInfo(pod),
		Timestamp:               now,
		InitialAttemptTimestamp: now,
		UnschedulablePlugins:    sets.NewString(unschedulableplugins...),
	}
}

// This is achieved by looking up the global clusterEventMap registry.
func (s *SchedulingQueue) podMatchesEvent(podInfo *framework.QueuedPodInfo, clusterEvent framework.ClusterEvent) bool {
	if clusterEvent.IsWildCard() {
		return true
	}

	for evt, nameSet := range s.clusterEventMap {
		// Firstly verify if the two ClusterEvents match:
		// - either the registered event from plugin side is a WildCardEvent,
		// - or the two events have identical Resource fields and *compatible* ActionType.
		//   Note the ActionTypes don't need to be *identical*. We check if the ANDed value
		//   is zero or not. In this way, it's easy to tell Update&Delete is not compatible,
		//   but Update&All is.
		evtMatch := evt.IsWildCard() ||
			(evt.Resource == clusterEvent.Resource && evt.ActionType&clusterEvent.ActionType != 0)

		// Secondly verify the plugin name matches.
		// Note that if it doesn't match, we shouldn't continue to search.
		if evtMatch && intersect(nameSet, podInfo.UnschedulablePlugins) {
			return true
		}
	}

	return false
}

func intersect(x, y sets.String) bool {
	if len(x) > len(y) {
		x, y = y, x
	}
	for v := range x {
		if y.Has(v) {
			return true
		}
	}
	return false
}

// isPodBackingoff returns true if a pod is still waiting for its backoff timer.
// If this returns true, the pod should not be re-tried.
func isPodBackingoff(podInfo *framework.QueuedPodInfo) bool {
	boTime := getBackoffTime(podInfo)
	return boTime.After(time.Now())
}

// getBackoffTime returns the time that podInfo completes backoff
func getBackoffTime(podInfo *framework.QueuedPodInfo) time.Time {
	duration := calculateBackoffDuration(podInfo)
	backoffTime := podInfo.Timestamp.Add(duration)
	return backoffTime
}

const (
	podInitialBackoffDuration = 1 * time.Second
	podMaxBackoffDuration     = 10 * time.Second
)

// calculateBackoffDuration is a helper function for calculating the backoffDuration
// based on the number of attempts the pod has made.
func calculateBackoffDuration(podInfo *framework.QueuedPodInfo) time.Duration {
	duration := podInitialBackoffDuration
	for i := 1; i < podInfo.Attempts; i++ {
		// Use subtraction instead of addition or multiplication to avoid overflow.
		if duration > podMaxBackoffDuration-duration {
			return podMaxBackoffDuration
		}
		duration += duration
	}
	return duration
}
