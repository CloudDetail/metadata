package export

import "github.com/CloudDetail/metadata/model/resource"

type ExporterType int

const (
	FlusherType      ExporterType = 0
	HTTPExporterType ExporterType = 0
)

var NonExporter = &Exporter{}

type Exporter struct {
	Exporters []resource.Exporter
}

func (e *Exporter) ExportResourceEvents(events *resource.ResourceEvent) {
	for _, exporter := range e.Exporters {
		exporter.ExportResourceEvents(events)
	}
}

func (e *Exporter) SetupResourcesRef(resources *resource.Resources) {
	for _, exporter := range e.Exporters {
		exporter.SetupResourcesRef(resources)
	}
}
