package flux

import "sync"

type State struct {
	mu    *sync.RWMutex
	store map[string]any

	// TODO: we can have a publisher here to notify manager about changing of state
}

func NewState() *State {
	return &State{
		mu:    new(sync.RWMutex),
		store: make(map[string]any),
	}
}

func (s *State) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO: notify manager about state change

	s.store[key] = value
}

func (s *State) Get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.store[key]
}
