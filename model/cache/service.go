package cache

import (
	"strings"
	"sync"

	"github.com/CloudDetail/metadata/model/resource"
)

var _ resource.ResHandler = &ServiceList{}

// ServiceList可以同时处理Service资源和Pod资源
// 如果同时处理Pod资源,会维护Service和Pod的关系
type ServiceList struct {
	*resource.Resources
	// Namespace/Name -> *Service
	ServiceMap sync.Map
	// IP -> *Service
	IP2ServiceMap sync.Map

	IsPodWatch bool
	// Namespace -> podMap,serviceMap
	// only enable when IsPodWatch is true
	nsScopePodServiceMap sync.Map
}

func NewServiceList(_ resource.ResType, resList []*resource.Resource) resource.ResHandler {
	sl := &ServiceList{
		Resources: &resource.Resources{
			ResList: resList,
			ResType: resource.ServiceType,
		},
		IsPodWatch: false,
	}

	if resList == nil {
		sl.Resources.ResList = []*resource.Resource{}
		return sl
	}

	for _, res := range resList {
		service := &Service{Resource: res}
		// 更新Service索引
		sl.updateServiceSearch(service)
	}
	return sl
}

func (sl *ServiceList) EnablePodMatch() {
	sl.IsPodWatch = true
}

func (sl *ServiceList) Reset(resList []*resource.Resource) {
	sl.ServiceMap = sync.Map{}
	sl.IP2ServiceMap = sync.Map{}

	newServiceMap := sync.Map{}
	newIP2ServiceMap := sync.Map{}

	for _, res := range resList {
		service := &Service{Resource: res}
		// 更新Service索引
		newServiceMap.Store(service.NS()+"/"+service.Name, service)
		newIP2ServiceMap.Store(service.IP(), service)
		// 接受Reset事件必然不是meta-agent, 不做 Pod/Service关系的处理
	}

	// !!! 这里的赋值包含了锁拷贝,
	// newxxxMap必须只在当前线程内使用 才是线程安全的
	sl.ServiceMap = newServiceMap
	sl.IP2ServiceMap = newIP2ServiceMap
	sl.Resources.Reset(resList)
}

// updateServiceSearch 更新Service资源常规的索引表
func (sl *ServiceList) updateServiceSearch(service *Service) {
	sl.ServiceMap.Store(service.NS()+"/"+service.Name, service)
	sl.IP2ServiceMap.Store(service.IP(), service)
}

func (sl *ServiceList) AddResource(res *resource.Resource) {
	switch res.ResType {
	case resource.PodType:
		if !sl.IsPodWatch {
			return
		}
		sl.addPod(res)
	case resource.ServiceType:
		service := &Service{
			Resource: res,
		}

		sl.updateServiceSearch(service)
		if sl.IsPodWatch {
			sl.checkRelation(service)
		}
		sl.Resources.AddResource(res)
	}
}

func (sl *ServiceList) addPod(res *resource.Resource) {
	pod := &Pod{
		Resource: res,
	}

	if pod.Phase() == "Pending" {
		// 跳过未就绪的Pod
		return
	}

	psMapRef, find := sl.nsScopePodServiceMap.Load(pod.NS())
	if !find {
		podServiceMap := &PodServiceMap{Namespace: pod.NS()}
		podServiceMap.PodMap.Store(pod.ResUID, pod)
		sl.nsScopePodServiceMap.Store(pod.NS(), podServiceMap)
		return
	}

	psMap := psMapRef.(*PodServiceMap)
	psMap.PodMap.Store(pod.ResUID, pod)
	psMap.ServiceMap.Range(func(_, serviceRef any) bool {
		service := serviceRef.(*Service)
		if service.MatchPod(pod) {
			service.AddEndpoint(pod)
			sl.Resources.UpdateResource(service.Resource)
		}
		return true
	})
}

func (sl *ServiceList) UpdateResource(res *resource.Resource) {
	switch res.ResType {
	case resource.PodType:
		if !sl.IsPodWatch {
			return
		}
		sl.updatePod(res)
	case resource.ServiceType:
		service := &Service{
			Resource: res,
		}

		oldServiceRef, find := sl.ServiceMap.Load(service.ResUID)
		if !find {
			sl.AddResource(res)
			return
		}

		oldService := oldServiceRef.(*Service)

		if oldService.IP() != service.IP() {
			sl.IP2ServiceMap.Delete(oldService.IP())
			sl.IP2ServiceMap.Store(service.IP(), service)
		}

		if sl.IsPodWatch {
			sl.checkRelation(service)
		}
		sl.Resources.UpdateResource(res)
	}
}

func (sl *ServiceList) updatePod(res *resource.Resource) {
	pod := &Pod{
		Resource: res,
	}

	psMapRef, find := sl.nsScopePodServiceMap.Load(pod.NS())
	if !find {
		if pod.Phase() != POD_PHASE_RUNNING {
			return
		}
		psMap := &PodServiceMap{Namespace: pod.NS()}
		psMap.PodMap.Store(pod.ResUID, pod)
		sl.nsScopePodServiceMap.Store(pod.NS(), psMap)
		return
	}

	psMap := psMapRef.(*PodServiceMap)
	var oldPodRef any
	if pod.Phase() != POD_PHASE_RUNNING {
		oldPodRef, find = psMap.PodMap.LoadAndDelete(pod.ResUID)
	} else {
		oldPodRef, find = psMap.PodMap.Swap(pod.ResUID, pod)
	}
	if !find {
		if pod.Phase() != POD_PHASE_RUNNING {
			return
		}
		psMap.ServiceMap.Range(func(_, serviceRef any) bool {
			service := serviceRef.(*Service)
			if service.MatchPod(pod) {
				service.AddEndpoint(pod)
				sl.Resources.UpdateResource(service.Resource)
			}
			return true
		})
		return
	}

	oldPod := oldPodRef.(*Pod)
	if oldPod.Phase() == POD_PHASE_RUNNING && pod.Phase() != POD_PHASE_RUNNING {
		// 查找Pod状态不再有效的Endpoint
		psMap.ServiceMap.Range(func(_, serviceRef any) bool {
			service := serviceRef.(*Service)
			if service.MatchPod(oldPod) {
				service.DeleteEndpoint(oldPod)
				sl.Resources.UpdateResource(service.Resource)
			}
			return true
		})
	} else {
		// 查找Label变化的Endpoint
		newLabels := pod.Labels()
		oldLabels := oldPod.Labels()
		if len(newLabels) == len(oldLabels) {
			var labelChanged = false
			for labelKey, labelValue := range oldLabels {
				if newLabels[labelKey] != labelValue {
					labelChanged = true
					break
				}
			}
			if !labelChanged {
				return
			}
		}
		psMap.ServiceMap.Range(func(_, serviceRef any) bool {
			service := serviceRef.(*Service)
			oldMatch := service.MatchPod(oldPod)
			if oldMatch && !service.MatchPod(pod) {
				service.DeleteEndpoint(oldPod)
				sl.Resources.UpdateResource(service.Resource)
			} else if !oldMatch && service.MatchPod(pod) {
				service.AddEndpoint(pod)
				sl.Resources.UpdateResource(service.Resource)
			}
			return true
		})
	}
}

func (sl *ServiceList) DeleteResource(res *resource.Resource) {
	switch res.ResType {
	case resource.PodType:
		if !sl.IsPodWatch {
			return
		}

		pod := &Pod{Resource: res}
		psMapRef, find := sl.nsScopePodServiceMap.Load(pod.NS())
		if !find {
			return
		}

		psMap := psMapRef.(*PodServiceMap)
		_, find = psMap.PodMap.LoadAndDelete(pod.ResUID)
		if !find {
			return
		}

		psMap.ServiceMap.Range(func(key, serviceRef any) bool {
			service := serviceRef.(*Service)
			if service.MatchPod(pod) {
				service.DeleteEndpoint(pod)
				sl.Resources.UpdateResource(service.Resource)
			}
			return true
		})
	case resource.ServiceType:
		service := &Service{
			Resource: res,
		}

		sl.Resources.DeleteResource(res)

		psMapRef, find := sl.nsScopePodServiceMap.Load(service.NS())
		if !find {
			return
		}
		psMap := psMapRef.(*PodServiceMap)
		psMap.ServiceMap.Delete(service.ResUID)
	}
}

func removeRelationByUID(relation []resource.Relation, relationType resource.RelationType, resourceUID resource.ResUID) []resource.Relation {
	for i, v := range relation {
		if v.ResUID == resourceUID && v.ReType == relationType {
			return append(relation[:i], relation[i+1:]...)
		}
	}
	return relation
}

func removeFromEndpoints(endpoints []string, oldEndpoint string) []string {
	for i, v := range endpoints {
		if v == oldEndpoint {
			return append(endpoints[:i], endpoints[i+1:]...)
		}
	}
	return endpoints
}

func (sl *ServiceList) checkRelation(service *Service) {
	psMapRef, find := sl.nsScopePodServiceMap.Load(service.NS())
	if !find {
		psMap := &PodServiceMap{Namespace: service.NS()}
		psMap.ServiceMap.Store(service.ResUID, service)
		sl.nsScopePodServiceMap.Store(service.NS(), psMap)
		return
	}
	psMap := psMapRef.(*PodServiceMap)
	psMap.ServiceMap.Store(service.ResUID, service)
	// 查询现有的Pod中能否匹配服务
	psMap.PodMap.Range(func(_, podRef any) bool {
		pod := podRef.(*Pod)
		if !service.MatchPod(pod) {
			return true
		}
		service.AddEndpoint(pod)
		return true
	})
}

type Service struct {
	*resource.Resource

	// only enable when isPodWatch = true
	endPoints []string
}

func (s *Service) UID() resource.ResUID {
	return s.ResUID
}

func (s *Service) IP() string {
	return s.StringAttr[resource.ServiceIP]
}

func (s *Service) EndPoints() []string {
	return strings.Split(s.StringAttr[resource.ServiceEndpoints], ",")
}

func (s *Service) AddEndpoint(pod *Pod) {
	// 不重复添加
	for _, relation := range s.Relations {
		if relation.ResUID == pod.ResUID {
			return
		}
	}
	s.endPoints = append(s.endPoints, pod.PodIP())
	s.Relations = append(s.Relations, resource.Relation{
		ResUID: pod.ResUID,
		ReType: resource.R_ENDPOINT,
		StringAttr: map[resource.AttrKey]string{
			resource.PodIP: pod.PodIP(),
		},
	})
	s.StringAttr[resource.ServiceEndpoints] = strings.Join(s.endPoints, ",")
}

func (s *Service) DeleteEndpoint(oldPod *Pod) {
	s.endPoints = removeFromEndpoints(s.endPoints, oldPod.PodIP())
	s.Relations = removeRelationByUID(s.Relations, resource.R_ENDPOINT, oldPod.ResUID)
	s.StringAttr[resource.ServiceEndpoints] = strings.Join(s.endPoints, ",")
}

func (s *Service) MatchedPods() []*resource.ResUID {
	podUIDs := []*resource.ResUID{}
	for _, relation := range s.Relations {
		if relation.ReType == resource.R_ENDPOINT {
			podUIDs = append(podUIDs, &relation.ResUID)
		}
	}
	return podUIDs
}

func (s *Service) Selectors() map[string]string {
	return s.Resource.ExtraAttr[resource.ServiceSelectorsAttr]
}

func (s *Service) NS() string {
	return s.Resource.StringAttr[resource.NamespaceAttr]
}

func (s *Service) MatchPod(pod *Pod) bool {
	if pod.Phase() == "Pending" {
		return false
	}

	serviceSelector := s.Selectors()
	podLabels := pod.Labels()
	if len(serviceSelector) == 0 || podLabels == nil {
		return false
	}
	for labelKey, labelValue := range s.Selectors() {
		value, find := podLabels[labelKey]
		if !find || value != labelValue {
			return false
		}
	}
	return true
}

type PodServiceMap struct {
	Namespace  string
	PodMap     sync.Map
	ServiceMap sync.Map
}
