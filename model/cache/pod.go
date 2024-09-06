package cache

import (
	"strings"
	"sync"

	"github.com/CloudDetail/metadata/model/resource"
)

var _ resource.ResHandler = &PodList{}

type PodList struct {
	*resource.Resources
	// POD UID -> *Pod
	UIDMap sync.Map
	// Namespace/Name -> *Pod
	PodMap sync.Map
	// ContainerID -> *Pod
	ContainerID2Pod sync.Map
	// IP -> *Pod only store not hostNetwork IP
	IP2PodMap sync.Map
}

func NewPodList(_ resource.ResType, resList []*resource.Resource) resource.ResHandler {
	pl := &PodList{
		Resources: &resource.Resources{
			ResType: resource.PodType,
			ResList: resList,
		},
	}

	if resList == nil {
		pl.Resources.ResList = []*resource.Resource{}
		return pl
	}

	// 重建查询表
	for _, res := range resList {
		pod := Pod{
			Resource: res,
		}
		pl.UIDMap.Store(pod.ResUID, &pod)
		pl.PodMap.Store(pod.NS()+"/"+pod.Name, &pod)
		pl.IP2PodMap.Store(pod.PodIP(), &pod)
		for _, containerID := range pod.ContainerIDs() {
			pl.ContainerID2Pod.Store(containerID, &pod)
		}
	}
	return pl
}

func (pl *PodList) Reset(resList []*resource.Resource) {
	newPodMap := sync.Map{}
	newContainerId2Pod := sync.Map{}
	newPodUIDMap := sync.Map{}
	newPodIP2Pod := sync.Map{}
	// 重建查询表
	for _, res := range resList {
		pod := Pod{
			Resource: res,
		}
		newPodUIDMap.Store(pod.ResUID, &pod)
		newPodMap.Store(pod.NS()+"/"+pod.Name, &pod)
		for _, containerID := range pod.ContainerIDs() {
			newContainerId2Pod.Store(containerID, &pod)
		}
		newPodIP2Pod.Store(pod.PodIP(), &pod)
	}

	// !!! 这里的赋值包含了锁拷贝,
	// newxxxMap必须只在当前线程内使用 才是线程安全的
	pl.PodMap = newPodMap
	pl.UIDMap = newPodUIDMap
	pl.ContainerID2Pod = newContainerId2Pod
	pl.IP2PodMap = newPodIP2Pod
	pl.Resources.Reset(resList)
}

func (pl *PodList) AddResource(res *resource.Resource) {
	pod := Pod{
		Resource: res,
	}

	pl.UIDMap.Store(pod.ResUID, &pod)
	pl.PodMap.Store(pod.NS()+"/"+pod.Name, &pod)
	for _, containerID := range pod.ContainerIDs() {
		pl.ContainerID2Pod.Store(containerID, &pod)
	}
	if !pod.IsHostNetWork() {
		pl.IP2PodMap.Store(pod.PodIP(), &pod)
	}
	pl.Resources.AddResource(res)
}

func (pl *PodList) UpdateResource(res *resource.Resource) {
	oldPod, find := pl.UIDMap.Load(res.ResUID)
	if find {
		for _, containerID := range oldPod.(*Pod).ContainerIDs() {
			pl.ContainerID2Pod.Delete(containerID)
		}
		pl.IP2PodMap.Delete(oldPod.(*Pod).PodIP())
	}

	newPod := Pod{
		Resource: res,
	}
	for _, containerID := range newPod.ContainerIDs() {
		pl.ContainerID2Pod.Store(containerID, &newPod)
	}
	if !newPod.IsHostNetWork() {
		pl.IP2PodMap.Store(newPod.PodIP(), &newPod)
	}
	pl.UIDMap.Store(newPod.ResUID, &newPod)
	pl.PodMap.Store(newPod.NS()+"/"+newPod.Name, &newPod)
	pl.Resources.UpdateResource(res)
}

func (pl *PodList) DeleteResource(res *resource.Resource) {
	oldPodRef, find := pl.UIDMap.LoadAndDelete(res.ResUID)
	if !find {
		return
	}
	oldPod := oldPodRef.(*Pod)
	pl.IP2PodMap.Delete(oldPod.PodIP())
	pl.UIDMap.Delete(oldPod.ResUID)
	pl.PodMap.Delete(oldPod.NS() + "/" + oldPod.Name)

	for _, containerID := range oldPod.ContainerIDs() {
		pl.ContainerID2Pod.Delete(containerID)
	}

	pl.Resources.DeleteResource(res)
}

type Pod struct {
	*resource.Resource
}

func (p *Pod) NS() string {
	return p.StringAttr[resource.NamespaceAttr]
}

func (p *Pod) ContainerIDs() []string {
	containerIds := p.StringAttr[resource.ContainerIDsAttr]
	if len(containerIds) == 0 {
		return []string{}
	}

	return strings.Split(containerIds, ",")
}

func (p *Pod) PodIP() string {
	return p.StringAttr[resource.PodIP]
}

func (p *Pod) Labels() map[string]string {
	return p.ExtraAttr[resource.PodLabelsAttr]
}

const (
	POD_PHASE_RUNNING = "Running"
	POD_PHASE_PENDING = "Pending"
)

func (p *Pod) Phase() string {
	return p.StringAttr[resource.PodPhase]
}

func (p *Pod) NodeName() string {
	return p.StringAttr[resource.PodHostName]
}

func (p *Pod) HostIP() string {
	return p.StringAttr[resource.PodHostIP]
}

func (p *Pod) IsHostNetWork() bool {
	val, find := p.Int64Attr[resource.PodHostNetwork]
	if !find {
		return false
	}
	return val != 0 // val == 0 -> false ; value != 0 -> true
}

type OwnerReferences struct {
	UID  string
	Kind string
	Name string
}

func (p *Pod) GetOwnerReferences(guessDeployment bool) []OwnerReferences {
	var ownerRefs []OwnerReferences
	for _, relation := range p.Relations {
		if relation.ReType != resource.R_OWNER {
			continue
		}

		ownerKind := relation.StringAttr[resource.OwnerType]
		ownerName := relation.StringAttr[resource.OwnerName]
		if ownerKind == "ReplicaSet" && guessDeployment {
			// TODO 如果监控了ReplicaSet信息,对RS进一步解析
			// 否则使用推测的Deployment信息
			ownerKind = "Deployment"
			lastPartIndex := strings.LastIndex(ownerName, "-")
			if lastPartIndex > 0 && lastPartIndex < len(ownerName) {
				ownerName = ownerName[:lastPartIndex]
			}
		}
		ownerRefs = append(ownerRefs, OwnerReferences{
			UID:  string(relation.ResUID),
			Kind: ownerKind,
			Name: ownerName,
		})
	}
	return ownerRefs
}
