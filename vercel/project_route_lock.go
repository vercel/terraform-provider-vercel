package vercel

import "sync"

var projectRouteLocks = newKeyedMutex()

type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{
		locks: map[string]*sync.Mutex{},
	}
}

func (m *keyedMutex) Lock(key string) func() {
	m.mu.Lock()
	lock, ok := m.locks[key]
	if !ok {
		lock = &sync.Mutex{}
		m.locks[key] = lock
	}
	m.mu.Unlock()

	lock.Lock()
	return lock.Unlock
}
