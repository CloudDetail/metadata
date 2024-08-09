package metasource

import (
	"github.com/CloudDetail/metadata/model/cache"
	"github.com/CloudDetail/metadata/model/resource"
)

type ClusterHandlerMap struct {
	ClusterID string
	exporter  resource.Exporter

	cache.HandlerMap
}

func (chm *ClusterHandlerMap) HandlerEvent(event *resource.ResourceEvent) {
	handler, find := chm.GetHandler(event.ResourceType)
	if !find {
		// create default handler, only used for query and transport to next meta source
		handler = resource.NewResources(event.ResourceType, []*resource.Resource{})
		handler.SetExporter(chm.exporter)
		handler.SetClusterID(chm.ClusterID)
		chm.AddHandler(event.ResourceType, handler)
	}
	switch event.Operation {
	case resource.AddOP:
		handler.AddResource(event.Res[0])
	case resource.UpdateOP:
		handler.UpdateResource(event.Res[0])
	case resource.DeleteOP:
		handler.DeleteResource(event.Res[0])
	case resource.ResetOP:
		handler.Reset(event.Res)
	}
}
