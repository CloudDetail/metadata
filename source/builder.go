package source

import (
	"fmt"
	"net/http"

	"github.com/CloudDetail/metadata/configs"
	"github.com/CloudDetail/metadata/export"
	"github.com/CloudDetail/metadata/model/cache"
	"github.com/CloudDetail/metadata/model/resource"
	"github.com/CloudDetail/metadata/server"
	"github.com/CloudDetail/metadata/source/apiserver"
	"github.com/CloudDetail/metadata/source/metasource"
)

type MetaSource interface {
	Run() error
	Stop() error

	// 用于将服务方法注册到外部的http服务中
	Handlers() map[string]http.HandlerFunc
}

type SourceType int

const (
	KubeAuth SourceType = iota
	AcceptPush
	Fetch
)

func CreateMetaSourceFromConfig(config *configs.MetaSourceConfig) MetaSource {
	var source MetaSource
	if config.KubeSource != nil {
		source = BuildKubeSource(config)
	} else {
		source = BuildMetaSource(config)
	}
	return source
}

func BuildKubeSource(config *configs.MetaSourceConfig) *apiserver.Watchers {
	var httpServer *server.HTTPServer
	if config.HttpServer != nil {
		httpServer = server.NewHTTPServer(fmt.Sprintf(":%d", config.HttpServer.Port))
	} else {
		httpServer = server.NewHTTPServer("")
	}

	apiserver.K8sWatcher.K8sConfig = apiserver.APIConfig{
		AuthType:     apiserver.AuthType(config.KubeSource.KubeAuthType),
		AuthFilePath: config.KubeSource.KubeAuthConfig,
	}

	exporters := []resource.Exporter{}
	if config.Exporter != nil {
		if len(config.Exporter.RemoteWriteAddr) > 0 {
			httpExporter := export.NewHTTPExporter(config.Exporter.RemoteWriteAddr)
			exporters = append(exporters, httpExporter)
		}
		// Deprecated
		if config.Exporter.FetchServerPort > 0 {
			fetchServer := export.NewFetcherServer()
			exporters = append(exporters, fetchServer)

			httpServer.SetListenAddr(fmt.Sprintf(":%d", config.Exporter.FetchServerPort))
			httpServer.RegisterHandler("/fetch", fetchServer.FetchWithWS)
		} else if config.Exporter.EnableFetchServer {
			fetchServer := export.NewFetcherServer()
			exporters = append(exporters, fetchServer)

			httpServer.RegisterHandler("/fetch", fetchServer.FetchWithWS)
		}
	}

	podList := cache.NewPodList(resource.PodType, nil)
	serviceList := cache.NewServiceList(resource.ServiceType, nil)
	nodeList := cache.NewNodeList(resource.NodeType, nil)

	if config.Querier != nil {
		cacheList := cache.NewSingleClusterCacheList()
		cache.SetupCacheMap(cacheList)

		// Deprecated
		if config.Querier.QueryServerPort > 0 {
			httpServer.SetListenAddr(fmt.Sprintf(":%d", config.Querier.QueryServerPort))
			httpServer.RegisterHandler("/query", cache.QueryInterface.QueryResource)
		} else if config.Querier.EnableQueryServer {
			httpServer.RegisterHandler("/query", cache.QueryInterface.QueryResource)
		}

		cacheList.AddResHandler("", resource.PodType, podList)
		cacheList.AddResHandler("", resource.ServiceType, serviceList)
		cacheList.AddResHandler("", resource.NodeType, nodeList)
	}

	if config.KubeSource.IsEndpointsNeeded {
		// ServiceList同时处理Service和Pod资源,构造关联关系
		serviceList.(*cache.ServiceList).EnablePodMatch()
		apiserver.K8sWatcher.WithHandler(resource.PodType, serviceList)
	}

	if len(config.KubeSource.ClusterID) > 0 {
		apiserver.K8sWatcher.ClusterID = config.KubeSource.ClusterID
	}

	return apiserver.K8sWatcher.
		WithHandler(resource.PodType, podList).
		WithHandler(resource.ServiceType, serviceList).
		WithHandler(resource.NodeType, nodeList).
		WithHttpServer(httpServer).
		WithExporters(exporters...)
}

func BuildMetaSource(config *configs.MetaSourceConfig) *metasource.MetaSource {
	var httpServer *server.HTTPServer
	if config.HttpServer != nil {
		httpServer = server.NewHTTPServer(fmt.Sprintf(":%d", config.HttpServer.Port))
	} else {
		httpServer = server.NewHTTPServer("")
	}

	exporters := []resource.Exporter{}
	if config.Exporter != nil {
		if len(config.Exporter.RemoteWriteAddr) > 0 {
			httpExporter := export.NewHTTPExporter(config.Exporter.RemoteWriteAddr)
			exporters = append(exporters, httpExporter)
		}

		// Deprecated
		if config.Exporter.FetchServerPort > 0 {
			fetchServer := export.NewFetcherServer()
			exporters = append(exporters, fetchServer)
			httpServer.RegisterHandler("/fetch", fetchServer.FetchWithWS)

			httpServer.SetListenAddr(fmt.Sprintf(":%d", config.Exporter.FetchServerPort))
		} else if config.Exporter.EnableFetchServer {
			fetchServer := export.NewFetcherServer()
			exporters = append(exporters, fetchServer)
			httpServer.RegisterHandler("/fetch", fetchServer.FetchWithWS)
		}
	}

	var cacheMap cache.CacheMap
	if config.Querier != nil {
		if config.Querier.IsSingleCluster {
			cacheMap = cache.NewSingleClusterCacheList()
		} else {
			cacheMap = cache.NewClusterCacheList()
		}
		cache.SetupCacheMap(cacheMap)

		// Deprecated
		if config.Querier.QueryServerPort > 0 {
			httpServer.SetListenAddr(fmt.Sprintf(":%d", config.Querier.QueryServerPort))
			httpServer.RegisterHandler("/query", cache.QueryInterface.QueryResource)
		} else if config.Querier.EnableQueryServer {
			httpServer.RegisterHandler("/query", cache.QueryInterface.QueryResource)
		}
	}

	return metasource.NewMetaSource().
		WithConfig(config).
		WithHandlerTemp(resource.PodType, cache.NewPodList).
		WithHandlerTemp(resource.ServiceType, cache.NewServiceList).
		WithHandlerTemp(resource.NodeType, cache.NewNodeList).
		WithHttpServer(httpServer).
		WithQuerier(cacheMap).
		WithExporters(exporters...)
}
