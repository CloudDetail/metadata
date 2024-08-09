package apiserver

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/CloudDetail/metadata/export"
	"github.com/CloudDetail/metadata/model/resource"
	"github.com/CloudDetail/metadata/server"
	"k8s.io/client-go/informers"
)

var K8sWatcher Watchers = Watchers{
	Watchers:       map[resource.ResType]IWatcher{},
	HandlerMap:     map[resource.ResType][]resource.ResHandler{},
	ExportResource: export.NonExporter,

	startedWatcher: []IWatcher{},
}

type ResourceHandlersMap map[resource.ResType][]resource.ResHandler

type Watchers struct {
	ctx    context.Context
	cancel context.CancelFunc

	Watchers   map[resource.ResType]IWatcher
	HandlerMap ResourceHandlersMap

	startedWatcher []IWatcher

	K8sConfig APIConfig
	ClusterID string

	HttpServer     *server.HTTPServer
	ExportResource resource.Exporter
}

func (w *Watchers) Handlers() map[string]http.HandlerFunc {
	if w.HttpServer == nil {
		return nil
	}
	return w.HttpServer.HandlerMap
}

func (w *Watchers) WithExporter(exportResource resource.Exporter) *Watchers {
	w.ExportResource = exportResource
	return w
}

func (w *Watchers) WithExporters(exporters ...resource.Exporter) *Watchers {
	w.ExportResource = &export.Exporter{
		Exporters: exporters,
	}
	return w
}

func (w *Watchers) WithHandler(resType resource.ResType, handler resource.ResHandler) *Watchers {
	if _, ok := w.HandlerMap[resType]; !ok {
		w.HandlerMap[resType] = make([]resource.ResHandler, 0)
	}
	w.HandlerMap[resType] = append(w.HandlerMap[resType], handler)
	return w
}

func (w *Watchers) WithHttpServer(s *server.HTTPServer) *Watchers {
	w.HttpServer = s
	return w
}

func (w *Watchers) Run() error {
	w.ctx = context.Background()
	clientSet, clusterIDFromAPIFingerprint, err := initClientSet(string(w.K8sConfig.AuthType), w.K8sConfig.AuthFilePath)
	if err != nil {
		return err
	}

	if len(w.ClusterID) == 0 {
		w.ClusterID = os.Getenv("CLUSTER_ID")
		if len(w.ClusterID) == 0 {
			w.ClusterID = clusterIDFromAPIFingerprint
		}
	}

	for resType, handlers := range w.HandlerMap {
		for _, handler := range handlers {
			handler.SetClusterID(w.ClusterID)
			handler.SetExporter(w.ExportResource)
		}
		if watcher, find := w.Watchers[resType]; find {
			w.startedWatcher = append(w.startedWatcher, watcher)
		}
	}

	factory := informers.NewSharedInformerFactory(clientSet, 10*time.Minute)
	for _, watcher := range w.startedWatcher {
		watcher.Init(w.ctx, clientSet, factory, "", w.HandlerMap)
		watcher.Run()
	}
	factory.Start(w.ctx.Done())
	factory.WaitForCacheSync(w.ctx.Done())

	return w.HttpServer.StartHttpServer()
}

func (w *Watchers) Stop() error {
	err := w.HttpServer.Stop()
	w.ctx.Done()
	return err
}

func addWatcher(resType resource.ResType, watcher IWatcher) {
	K8sWatcher.Watchers[resType] = watcher
}
