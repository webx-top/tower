package com

import (
	"sync"
)

func InitSafeMap[K comparable, V any]() SafeMap[K, V] {
	return SafeMap[K, V]{
		lock: new(sync.RWMutex),
		bm:   make(map[K]V),
	}
}

func NewSafeMap[K comparable, V any]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		lock: new(sync.RWMutex),
		bm:   make(map[K]V),
	}
}

type SafeMap[K comparable, V any] struct {
	lock *sync.RWMutex
	bm   map[K]V
}

func (m *SafeMap[K, V]) Size() int {
	m.lock.RLock()
	size := len(m.bm)
	m.lock.RUnlock()
	return size
}

func (m *SafeMap[K, V]) GetOk(k K) (V, bool) {
	m.lock.RLock()
	r, y := m.bm[k]
	m.lock.RUnlock()
	return r, y
}

// Get from maps return the k's value
func (m *SafeMap[K, V]) Get(k K) V {
	m.lock.RLock()
	r := m.bm[k]
	m.lock.RUnlock()
	return r
}

func (m *SafeMap[K, V]) Gets(keys ...K) []V {
	m.lock.RLock()
	res := make([]V, 0, len(keys))
	for _, key := range keys {
		val, ok := m.bm[key]
		if ok {
			res = append(res, val)
		}
	}
	m.lock.RUnlock()
	return res
}

func (m *SafeMap[K, V]) Remove(keys ...K) {
	m.lock.Lock()
	for _, key := range keys {
		delete(m.bm, key)
	}
	m.lock.Unlock()
}

func (m *SafeMap[K, V]) Range(f func(key K, val V) bool) {
	m.lock.RLock()
	for key, val := range m.bm {
		if !f(key, val) {
			break
		}
	}
	m.lock.RUnlock()
}

func (m *SafeMap[K, V]) ClearEmpty(f func(key K, val V) bool) {
	m.lock.Lock()
	for key, val := range m.bm {
		if f(key, val) {
			delete(m.bm, key)
		}
	}
	m.lock.Unlock()
}

func (m *SafeMap[K, V]) Reset() {
	m.lock.Lock()
	clear(m.bm)
	m.lock.Unlock()
}

// Set maps the given key and value. Returns false
// if the key is already in the map and changes nothing.
func (m *SafeMap[K, V]) Set(k K, v V) {
	m.lock.Lock()
	m.bm[k] = v
	m.lock.Unlock()
}

// Exists returns true if k is exist in the map.
func (m *SafeMap[K, V]) Exists(k K) bool {
	m.lock.RLock()
	_, ok := m.bm[k]
	m.lock.RUnlock()
	return ok
}

func (m *SafeMap[K, V]) Delete(k K) {
	m.lock.Lock()
	delete(m.bm, k)
	m.lock.Unlock()
}

func (m *SafeMap[K, V]) Items() map[K]V {
	m.lock.RLock()
	r := m.bm
	m.lock.RUnlock()
	return r
}

func InitOrderlySafeMap[K comparable, V any]() OrderlySafeMap[K, V] {
	return OrderlySafeMap[K, V]{
		SafeMap: NewSafeMap[K, V](),
		keys:    []K{},
	}
}

func NewOrderlySafeMap[K comparable, V any]() *OrderlySafeMap[K, V] {
	return &OrderlySafeMap[K, V]{
		SafeMap: NewSafeMap[K, V](),
		keys:    []K{},
	}
}

type OrderlySafeMap[K comparable, V any] struct {
	*SafeMap[K, V]
	keys []K // map keys
}

func (m *OrderlySafeMap[K, V]) Set(k K, v V) {
	m.lock.Lock()
	if _, ok := m.bm[k]; !ok {
		m.bm[k] = v
		m.keys = append(m.keys, k)
	} else {
		m.bm[k] = v
	}
	m.lock.Unlock()
}

func (m *OrderlySafeMap[K, V]) Delete(k K) {
	m.lock.Lock()
	delete(m.bm, k)
	m.removeKey(k)
	m.lock.Unlock()
}

func (m *OrderlySafeMap[K, V]) removeKey(k K) {
	endIndex := len(m.keys) - 1
	for index, mapKey := range m.keys {
		if mapKey != k {
			continue
		}
		if index == endIndex {
			m.keys = m.keys[0:index]
			break
		}
		if index == 0 {
			m.keys = m.keys[1:]
			break
		}
		m.keys = append(m.keys[0:index], m.keys[index+1:]...)
		return
	}
}

func (m *OrderlySafeMap[K, V]) Remove(keys ...K) {
	m.lock.Lock()
	for _, k := range keys {
		delete(m.bm, k)
		m.removeKey(k)
	}
	m.lock.Unlock()
}

func (m *OrderlySafeMap[K, V]) Keys() []K {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.keys
}

func (m *OrderlySafeMap[K, V]) ClearEmpty(f func(key K, val V) bool) {
	m.lock.Lock()
	for key, val := range m.bm {
		if f(key, val) {
			delete(m.bm, key)
			m.removeKey(key)
		}
	}
	m.lock.Unlock()
}

func (m *OrderlySafeMap[K, V]) Reset() {
	m.lock.Lock()
	clear(m.bm)
	clear(m.keys)
	m.lock.Unlock()
}

func (m *OrderlySafeMap[K, V]) Values(force ...bool) []V {
	m.lock.RLock()
	values := make([]V, 0, len(m.keys))
	for _, mapKey := range m.keys {
		values = append(values, m.bm[mapKey])
	}
	m.lock.RUnlock()
	return values
}

func (m *OrderlySafeMap[K, V]) VisitAll(callback func(int, K, V)) {
	m.lock.RLock()
	for index, mapKey := range m.keys {
		callback(index, mapKey, m.bm[mapKey])
	}
	m.lock.RUnlock()
}
