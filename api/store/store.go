package store

import (
	"sync"

	"github.com/ixcsoft/idp/shared/queue"
)

type Store struct {
	states sync.Map // deploy_id -> queue.StatusEvent
}

func New() *Store { return &Store{} }

func (s *Store) Set(ev queue.StatusEvent) {
	s.states.Store(ev.DeployID, ev)
}

func (s *Store) Get(deployID string) (queue.StatusEvent, bool) {
	v, ok := s.states.Load(deployID)
	if !ok {
		return queue.StatusEvent{}, false
	}
	return v.(queue.StatusEvent), true
}
