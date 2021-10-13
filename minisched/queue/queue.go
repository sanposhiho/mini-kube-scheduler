package queue

import (
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
)

type SchedulingQueue struct {
	activeQ []*v1.Pod
	lock    sync.RWMutex
}

func New() *SchedulingQueue {
	return &SchedulingQueue{
		activeQ: []*v1.Pod{},
	}
}

func (s *SchedulingQueue) Add(pod *v1.Pod) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	
	s.activeQ = append(s.activeQ, pod)
	return nil
}

func (s *SchedulingQueue) NextPod() *v1.Pod {
	// wait
	for len(s.activeQ) == 0 {
	}

	p := s.activeQ[0]
	s.activeQ = s.activeQ[1:]
	return p
}
