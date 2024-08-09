package metasource

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/CloudDetail/metadata/configs"
	"github.com/CloudDetail/metadata/export"
	"github.com/CloudDetail/metadata/model/cache"
	"github.com/CloudDetail/metadata/model/resource"
	"github.com/CloudDetail/metadata/server"
	lru "github.com/hashicorp/golang-lru/v2"
)

type MetaSource struct {
	ctx                context.Context
	cfg                *configs.MetaSourceConfig
	HandlerTemplateMap map[resource.ResType]resource.HandlerTemplate
	// clusterId(string) -> *cache.ClusterHandlerMap
	ClusterMaps sync.Map

	Exporter        resource.Exporter
	QuerierCacheMap cache.CacheMap
	HttpServer      *server.HTTPServer

	AgentLastCheckPoint *lru.Cache[int64, *resource.CheckPoint]
	AgentCounter        atomic.Int64
	stop                func() error
}

func (r *MetaSource) Handlers() map[string]http.HandlerFunc {
	if r.HttpServer == nil {
		return nil
	}
	return r.HttpServer.HandlerMap
}

func NewMetaSource() *MetaSource {
	agentMap, _ := lru.New[int64, *resource.CheckPoint](1000)
	return &MetaSource{
		ctx:                 context.Background(),
		HandlerTemplateMap:  map[resource.ResType]resource.HandlerTemplate{},
		ClusterMaps:         sync.Map{},
		Exporter:            export.NonExporter,
		AgentLastCheckPoint: agentMap,
		stop:                func() error { return nil },
	}
}

func (s *MetaSource) WithConfig(cfg *configs.MetaSourceConfig) *MetaSource {
	s.cfg = cfg
	return s
}

func (s *MetaSource) WithQuerier(querier cache.CacheMap) *MetaSource {
	s.QuerierCacheMap = querier
	return s
}

func (s *MetaSource) WithHandlerTemp(resType resource.ResType, handler resource.HandlerTemplate) *MetaSource {
	s.HandlerTemplateMap[resType] = handler
	return s
}

func (s *MetaSource) WithExporter(exporter resource.Exporter) *MetaSource {
	s.Exporter = exporter
	return s
}

func (s *MetaSource) WithExporters(exporters ...resource.Exporter) *MetaSource {
	s.Exporter = &export.Exporter{
		Exporters: exporters,
	}
	return s
}

func (s *MetaSource) WithHttpServer(srv *server.HTTPServer) *MetaSource {
	s.HttpServer = srv
	return s
}

func (s *MetaSource) Stop() error {
	s.HttpServer.Stop()
	return s.stop()
}

func (s *MetaSource) Run() error {
	if s.cfg.AcceptEventSource != nil {
		// Deprecated
		if s.cfg.AcceptEventSource.AcceptEventPort > 0 {
			s.HttpServer.SetListenAddr(fmt.Sprintf(":%d", s.cfg.AcceptEventSource.AcceptEventPort))
			s.HttpServer.RegisterHandler("/push", s.HandlePushedEvent)
		} else if s.cfg.AcceptEventSource.EnableAcceptServer {
			s.HttpServer.RegisterHandler("/push", s.HandlePushedEvent)
		}
	} else if s.cfg.FetchSource != nil {
		go s.RunWithFetcher(s.cfg.FetchSource.SourceAddr)
	} else {
		return fmt.Errorf("invalid meta source config")
	}

	return s.HttpServer.StartHttpServer()
}

func (s *MetaSource) initClusterHandlerMap(
	clusterId string,
) *ClusterHandlerMap {
	handlersMap := &ClusterHandlerMap{
		ClusterID: clusterId,
		exporter:  s.Exporter,
		HandlerMap: cache.HandlerMap{
			Handlers: map[resource.ResType]resource.ResHandler{},
		},
	}

	// 根据发来的数据和注册的处理模版进行初始化
	for resType, temp := range s.HandlerTemplateMap {
		handler := temp(resType, nil)
		handler.SetClusterID(clusterId)
		handler.SetExporter(s.Exporter)
		handlersMap.AddHandler(resType, handler)
	}

	if s.QuerierCacheMap != nil {
		s.QuerierCacheMap.AddResHandlers(clusterId, &handlersMap.HandlerMap)
	}

	return handlersMap
}
