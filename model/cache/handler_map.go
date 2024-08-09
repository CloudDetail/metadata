package cache

import (
	"sync"

	"github.com/CloudDetail/metadata/model/resource"
)

type HandlerMap struct {
	Handlers map[resource.ResType]resource.ResHandler

	sync.RWMutex
}

func (m *HandlerMap) GetHandler(resType resource.ResType) (resource.ResHandler, bool) {
	m.RLock()
	defer m.RUnlock()
	handler, find := m.Handlers[resType]
	return handler, find
}

func (m *HandlerMap) AddHandler(resType resource.ResType, handler resource.ResHandler) {
	m.Lock()
	defer m.Unlock()
	m.Handlers[resType] = handler
}
