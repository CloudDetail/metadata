package cache

import (
	"encoding/json"
	"net/http"

	"github.com/CloudDetail/metadata/model/resource"
)

var Querier = &Query{
	CacheMap: NonCache,
}
var QueryInterface IQuery = Querier

type IQuery interface {
	SetCacheMap(cacheMap CacheMap)
	QueryResource(w http.ResponseWriter, r *http.Request)
}

type Query struct {
	CacheMap
}

func SetupCacheMap(cacheMap CacheMap) {
	QueryInterface.SetCacheMap(cacheMap)
}

type QueryResRequest struct {
	ClusterID    string
	ResType      resource.ResType
	ResName      string
	ResNamespace string
	IP           string
	ListAll      bool
}

type ResInfo struct {
	IsFind bool
	Object any
}

func (q *Query) SetCacheMap(cacheMap CacheMap) {
	q.CacheMap = cacheMap
}

func (q *Query) QueryResource(w http.ResponseWriter, r *http.Request) {
	var req QueryResRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return
	}
	defer r.Body.Close()

	resp := &ResInfo{
		IsFind: false,
		Object: nil,
	}
	if req.ListAll {
		resp.IsFind = true
		if req.ResType == resource.PodType {
			resp.Object = q.ListPod(req.ClusterID)
		} else if req.ResType == resource.ServiceType {
			resp.Object = q.ListService(req.ClusterID)
		}
	} else {
		if req.ResType == resource.PodType {
			resp.Object, resp.IsFind = q.GetPodByNSAndName(req.ClusterID, req.ResNamespace, req.ResName)
		} else {
			resp.Object, resp.IsFind = q.GetServiceByIP(req.ClusterID, req.IP)
		}
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (q *Query) GetPodByContainerId(clusterID string, containerId string) (*Pod, bool) {
	if len(containerId) > 12 {
		containerId = containerId[:12]
	}

	if len(clusterID) == 0 {
		handlers, find := q.GetCaches(resource.PodType)
		if !find {
			return nil, false
		}
		for _, handler := range handlers {
			if podRef, find := handler.(*PodList).ContainerID2Pod.Load(containerId); find {
				return podRef.(*Pod), find
			}
		}
		return nil, false
	}
	if handler, find := q.GetCache(clusterID, resource.PodType); find {
		podRef, find := handler.(*PodList).ContainerID2Pod.Load(containerId)
		return podRef.(*Pod), find
	}

	return nil, false
}

func (q *Query) GetPodByNSAndName(clusterID string, namespace string, name string) (*Pod, bool) {
	// TODO
	return nil, false
}

func (q *Query) GetPodByUID(clusterID string, UID resource.ResUID) (*Pod, bool) {
	if handler, find := q.GetCache(clusterID, resource.PodType); find {
		podRef, find := handler.(*PodList).PodMap.Load(UID)
		return podRef.(*Pod), find
	}

	return nil, false
}

func (q *Query) ListService(clusterID string) (services []*Service) {
	if len(clusterID) == 0 {
		handlers, find := q.GetCaches(resource.ServiceType)
		if !find {
			return nil
		}

		for _, handler := range handlers {
			handler.(*ServiceList).ServiceMap.Range(func(_, serviceRef interface{}) bool {
				services = append(services, serviceRef.(*Service))
				return true
			})
		}
		return services
	}
	if handler, find := q.GetCache(clusterID, resource.PodType); find {
		handler.(*PodList).PodMap.Range(func(_, serviceRef interface{}) bool {
			services = append(services, serviceRef.(*Service))
			return true
		})
		return services
	}
	return nil
}

func (q *Query) ListPod(clusterID string) (pods []*Pod) {
	if len(clusterID) == 0 {
		handlers, find := q.GetCaches(resource.PodType)
		if !find {
			return nil
		}
		for _, handler := range handlers {
			handler.(*PodList).PodMap.Range(func(_, podRef any) bool {
				pods = append(pods, podRef.(*Pod))
				return true
			})
		}
		return pods
	}
	if handler, find := q.GetCache(clusterID, resource.PodType); find {
		handler.(*PodList).PodMap.Range(func(_, podRef any) bool {
			pods = append(pods, podRef.(*Pod))
			return true
		})
		return pods
	}
	return nil
}

func (q *Query) GetServiceByIP(clusterID string, serviceIP string) (*Service, bool) {
	if len(clusterID) == 0 {
		handlers, find := q.GetCaches(resource.ServiceType)
		if !find {
			return nil, false
		}
		for _, handler := range handlers {
			if serviceRef, find := handler.(*ServiceList).IP2ServiceMap.Load(serviceIP); find {
				return serviceRef.(*Service), find
			}
		}
		return nil, false
	}
	if handler, find := q.GetCache(clusterID, resource.ServiceType); find {
		serviceRef, find := handler.(*ServiceList).IP2ServiceMap.Load(serviceIP)
		return serviceRef.(*Service), find
	}

	return nil, false
}

func (q *Query) GetPodByIP(clusterID string, podIP string) (*Pod, bool) {
	if len(clusterID) == 0 {
		handlers, find := q.GetCaches(resource.PodType)
		if !find {
			return nil, false
		}
		for _, handler := range handlers {
			if podRef, find := handler.(*PodList).IP2PodMap.Load(podIP); find {
				return podRef.(*Pod), find
			}
		}
		return nil, false
	}
	if handler, find := q.GetCache(clusterID, resource.PodType); find {
		podRef, find := handler.(*PodList).IP2PodMap.Load(podIP)
		return podRef.(*Pod), find
	}
	return nil, false
}

func (q *Query) GetNodeByIP(clusterID string, IP string) (*Node, bool) {
	if len(clusterID) == 0 {
		handlers, find := q.GetCaches(resource.NodeType)
		if !find {
			return nil, false
		}
		for _, handler := range handlers {
			if nodeRef, find := handler.(*NodeList).IP2Node.Load(IP); find {
				return nodeRef.(*Node), find
			}
		}
		return nil, false
	}
	if handler, find := q.GetCache(clusterID, resource.NodeType); find {
		nodeRef, find := handler.(*NodeList).IP2Node.Load(IP)
		return nodeRef.(*Node), find
	}
	return nil, false
}
