package apiserver

import (
	"context"
	"log"
	"testing"

	"github.com/CloudDetail/metadata/export"
	"github.com/CloudDetail/metadata/model/cache"
	"github.com/CloudDetail/metadata/model/resource"
)

func TestWatcher(t *testing.T) {
	K8sWatcher.K8sConfig = APIConfig{
		AuthType:     AuthTypeKubeConfig,
		AuthFilePath: "testdata/kubeconfig",
	}
	K8sWatcher.ctx = context.Background()

	httpExporter := export.NewHTTPExporter("http://localhost:8080")
	fetchExporter := export.NewFetcherServer()
	podList := cache.NewPodList(resource.PodType, nil)
	serviceList := cache.NewServiceList(resource.ServiceType, nil)

	querier := &cache.SingleClusterCacheMap{}
	querier.AddResHandler("", resource.PodType, podList)
	querier.AddResHandler("", resource.ServiceType, serviceList)

	watchers := K8sWatcher.
		WithHandler(resource.PodType, podList).
		WithHandler(resource.PodType, serviceList).
		WithHandler(resource.ServiceType, serviceList).
		WithExporters(httpExporter, fetchExporter)

	err := watchers.Run()
	if err != nil {
		log.Printf("watch k8s failed, err: %v", err)
	}

	select {}
}
