package cache

import (
	"sync"

	"github.com/CloudDetail/metadata/model/resource"
)

type CacheMap interface {
	AddResHandler(clusterId string, resType resource.ResType, handler resource.ResHandler)
	AddResHandlers(clusterId string, handlers *HandlerMap)

	GetCache(clusterId string, resType resource.ResType) (resource.ResHandler, bool)
	GetCaches(resType resource.ResType) ([]resource.ResHandler, bool)
}

var _ CacheMap = &ClusterCacheMap{}
var _ CacheMap = &SingleClusterCacheMap{}
var NonCache CacheMap = &NonCacheMap{}

type NonCacheMap struct{}

// GetCache implements CacheMap.
func (n *NonCacheMap) GetCache(clusterId string, resType resource.ResType) (resource.ResHandler, bool) {
	return nil, false
}

// GetCaches implements CacheMap.
func (n *NonCacheMap) GetCaches(resType resource.ResType) ([]resource.ResHandler, bool) {
	return nil, false
}

// AddResHandler implements CacheMap.
func (n *NonCacheMap) AddResHandler(clusterId string, resType resource.ResType, handler resource.ResHandler) {
}

// AddResHandlers implements CacheMap.
func (n *NonCacheMap) AddResHandlers(clusterId string, handlers *HandlerMap) {
}

type SingleClusterCacheMap struct {
	Handlers *HandlerMap
}

// GetCaches implements CacheMap.
func (b *SingleClusterCacheMap) GetCaches(resType resource.ResType) ([]resource.ResHandler, bool) {
	handler, bool := b.GetCache("", resType)
	return []resource.ResHandler{handler}, bool
}

func NewSingleClusterCacheList() *SingleClusterCacheMap {
	return &SingleClusterCacheMap{
		Handlers: &HandlerMap{
			Handlers: make(map[resource.ResType]resource.ResHandler),
		},
	}
}

// GetCache implements Querier.
func (b *SingleClusterCacheMap) GetCache(clusterId string, resType resource.ResType) (resource.ResHandler, bool) {
	handler, find := b.Handlers.GetHandler(resType)
	return handler, find
}

// AddResHandler implements Querier.z
func (b *SingleClusterCacheMap) AddResHandler(_ string, resType resource.ResType, handler resource.ResHandler) {
	b.Handlers.AddHandler(resType, handler)
}

// AddResHandlers implements Querier.
func (b *SingleClusterCacheMap) AddResHandlers(_ string, handlers *HandlerMap) {
	b.Handlers = handlers
}

type ClusterCacheMap struct {
	// ClusterID -> handlersMap
	Caches sync.Map
}

// GetCaches implements CacheMap.
func (b *ClusterCacheMap) GetCaches(resType resource.ResType) ([]resource.ResHandler, bool) {
	var handlers []resource.ResHandler
	b.Caches.Range(func(key, value any) bool {
		if handler, find := value.(*HandlerMap).GetHandler(resType); find {
			handlers = append(handlers, handler)
		}
		return true
	})

	return handlers, true
}

func NewClusterCacheList() *ClusterCacheMap {
	return &ClusterCacheMap{}
}

// AddResHandler implements Querier.
func (b *ClusterCacheMap) AddResHandler(clusterId string, resType resource.ResType, handler resource.ResHandler) {
	handlerMap, find := b.Caches.Load(clusterId)
	if !find {
		b.Caches.Store(clusterId, &HandlerMap{})
	}

	handlerMap.(*HandlerMap).AddHandler(resType, handler)
	b.Caches.Store(clusterId, handlerMap)
}

// AddResHandlers implements Querier.
func (b *ClusterCacheMap) AddResHandlers(clusterId string, handlers *HandlerMap) {
	b.Caches.Store(clusterId, handlers)
}

func (b *ClusterCacheMap) GetCache(clusterId string, resType resource.ResType) (resource.ResHandler, bool) {
	handlerMap, find := b.Caches.Load(clusterId)
	if !find {
		return nil, false
	}
	handler, find := handlerMap.(*HandlerMap).GetHandler(resType)
	return handler, find
}
