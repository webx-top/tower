package rundelay

import (
	"sync"
	"time"
)

func NewMultiple[T any](delay time.Duration, f func(T) error, once ...bool) *Multiple[T] {
	m := &Multiple[T]{
		mp:    map[string]RunDelayer[T]{},
		mu:    sync.RWMutex{},
		delay: delay,
		exec:  f,
		once:  false,
	}
	if len(once) > 0 {
		m.once = once[0]
	}
	return m
}

type Multiple[T any] struct {
	mp    map[string]RunDelayer[T]
	mu    sync.RWMutex
	delay time.Duration
	exec  func(T) error
	once  bool
}

func (m *Multiple[T]) get(k string) (RunDelayer[T], bool) {
	m.mu.RLock()
	val, ok := m.mp[k]
	m.mu.RUnlock()
	return val, ok
}

func (m *Multiple[T]) set(k string, v RunDelayer[T]) {
	m.mu.Lock()
	m.mp[k] = v
	m.mu.Unlock()
}

func (m *Multiple[T]) Delete(k string) {
	if m.mu.TryLock() {
		delete(m.mp, k)
		m.mu.Unlock()
		return
	}

	delete(m.mp, k)
}

func (m *Multiple[T]) Init(delay time.Duration, f func(T) error) {
	m.exec = f
	m.delay = delay
}

func (m *Multiple[T]) Run(k string, v T) bool {
	m.mu.Lock()
	val, ok := m.mp[k]
	if !ok {
		val = New(m.delay, func(t T) (err error) {
			err = m.exec(t)
			if m.once {
				m.Delete(k)
			}
			return
		})
		m.mp[k] = val
	}
	m.mu.Unlock()
	return val.Run(v)
}

func (m *Multiple[T]) Done(k string) error {
	val, ok := m.get(k)
	if ok {
		return val.Done()
	}
	return nil
}

func (m *Multiple[T]) Close(k ...string) (err error) {
	if len(k) > 0 {
		m.CloseByKey(k...)
		return
	}
	m.mu.Lock()
	for k, v := range m.mp {
		err = v.Close()
		if err != nil {
			break
		}
		delete(m.mp, k)
	}
	m.mu.Unlock()
	return
}

func (m *Multiple[T]) CloseByKey(k ...string) {
	m.mu.Lock()
	for _, _k := range k {
		val, ok := m.mp[_k]
		if ok {
			val.Close()
		}
	}
	m.mu.Unlock()
}

func (m *Multiple[T]) Range(cb func(string, RunDelayer[T])) error {
	m.mu.RLock()
	for k, v := range m.mp {
		cb(k, v)
	}
	m.mu.RUnlock()
	return nil
}
