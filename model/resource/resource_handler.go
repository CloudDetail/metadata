package resource

type HandlerTemplate func(resType ResType, resources []*Resource) ResHandler

type ResHandler interface {
	AddResource(res *Resource)
	UpdateResource(res *Resource)
	DeleteResource(res *Resource)
	Reset(resList []*Resource)

	SetClusterID(clusterID string)
	SetExporter(Exporter)
}

type Exporter interface {
	SetupResourcesRef(resources *Resources)
	ExportResourceEvents(events *ResourceEvent)
}
