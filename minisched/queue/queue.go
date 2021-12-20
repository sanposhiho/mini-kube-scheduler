package queue

import (
	"sync"

	v1 "k8s.io/api/core/v1"
)

type SchedulingQueue struct {
	activeQ []*v1.Pod
	lock    *sync.Cond
}

func New() *SchedulingQueue {
	return &SchedulingQueue{
		activeQ: []*v1.Pod{},
		lock:    sync.NewCond(&sync.Mutex{}),
	}
}

func (s *SchedulingQueue) Add(pod *v1.Pod) {
	s.lock.L.Lock()
	defer s.lock.L.Unlock()

	s.activeQ = append(s.activeQ, pod)
	s.lock.Signal()
	return
}

func (s *SchedulingQueue) NextPod() *v1.Pod {
	// wait
	s.lock.L.Lock()
	for len(s.activeQ) == 0 {
		s.lock.Wait()
	}

	p := s.activeQ[0]
	s.activeQ = s.activeQ[1:]
	s.lock.L.Unlock()
	return p
}
