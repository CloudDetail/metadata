package resource

import (
	"log"
	"sync"
)

var _ ResHandler = &Resources{}

type Resources struct {
	ResList []*Resource
	ResType ResType

	ClusterID string

	// add lock/unlock...
	ExportMux sync.RWMutex
	Exporter
}

func NewResources(resType ResType, resList []*Resource) *Resources {
	resources := &Resources{
		ResList: resList,
		ResType: resType,
	}
	return resources
}

func (rs *Resources) SetClusterID(clusterID string) {
	rs.ClusterID = clusterID
}

func (rs *Resources) SetExporter(exporter Exporter) {
	// 不重复注册
	if rs.Exporter == nil {
		exporter.SetupResourcesRef(rs)
		rs.Exporter = exporter
	}
}

func (rs *Resources) updateResList(res *Resource) (isUpdated bool) {
	rs.ExportMux.Lock()
	defer rs.ExportMux.Unlock()
	for _, item := range rs.ResList {
		if item.ResUID == res.ResUID {
			item.ResVersion = res.ResVersion
			item.Relations = res.Relations
			item.StringAttr = res.StringAttr
			item.Int64Attr = res.Int64Attr
			return true
		}
	}
	rs.ResList = append(rs.ResList, res)
	return false
}
func (rs *Resources) UpdateResource(res *Resource) {
	isUpdate := rs.updateResList(res)

	var event *ResourceEvent
	if isUpdate {
		event = &ResourceEvent{
			ClusterID:    rs.ClusterID,
			Res:          []*Resource{res},
			ResourceType: res.ResType,
			Operation:    UpdateOP,
		}
	} else {
		event = &ResourceEvent{
			ClusterID:    rs.ClusterID,
			Res:          []*Resource{res},
			ResourceType: res.ResType,
			Operation:    AddOP,
		}
	}
	rs.ExportResourceEvents(event)
}

func (rs *Resources) Reset(res []*Resource) {
	rs.ResList = res

	log.Printf("reset resources: [%s](%d) and send reset event to exporter", rs.ClusterID, rs.ResType)
	rs.ExportResourceEvents(&ResourceEvent{
		ClusterID:    rs.ClusterID,
		Res:          res,
		ResourceType: rs.ResType,
		Operation:    ResetOP,
	})
}

func (rs *Resources) AddResource(res *Resource) {
	// 始终检查ResList中是否有相同UID的资源
	rs.updateResList(res)
	rs.ExportResourceEvents(&ResourceEvent{
		ClusterID:    rs.ClusterID,
		Res:          []*Resource{res},
		ResourceType: res.ResType,
		Operation:    AddOP,
	})
}

func (rs *Resources) deleteFromResList(res *Resource) (isDeleted bool) {
	rs.ExportMux.Lock()
	defer rs.ExportMux.Unlock()

	rs.ResList, isDeleted = removeElement(rs.ResList, res.ResUID)
	return isDeleted
}

func (rs *Resources) DeleteResource(res *Resource) {
	isDeleted := rs.deleteFromResList(res)
	if !isDeleted {
		return
	}
	rs.ExportResourceEvents(&ResourceEvent{
		ClusterID:    rs.ClusterID,
		Res:          []*Resource{res},
		ResourceType: res.ResType,
		Operation:    DeleteOP,
	})
}

func removeElement(slice []*Resource, UID ResUID) ([]*Resource, bool) {
	for i, v := range slice {
		if v.ResUID == UID {
			return append(slice[:i], slice[i+1:]...), true
		}
	}
	return slice, false
}
