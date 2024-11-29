package apiserver

import (
	"context"
	"strconv"
	"strings"

	"github.com/CloudDetail/metadata/model/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func init() {
	addWatcher(resource.PodType, &PodWatcher{})
}

type PodWatcher struct {
	ctx     context.Context
	client  *kubernetes.Clientset
	factory informers.SharedInformerFactory

	handlers  []resource.ResHandler
	namespace string
}

func (w *PodWatcher) Init(
	ctx context.Context,
	client *kubernetes.Clientset,
	factory informers.SharedInformerFactory,
	namespace string,
	handlersMap ResourceHandlersMap,
) {
	w.ctx = ctx
	w.client = client
	w.factory = factory
	w.namespace = namespace
	w.handlers = handlersMap[resource.PodType]
}

func (w *PodWatcher) Run() {
	informer := w.factory.Core().V1().Pods().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				podRes := createResourceFromPod(pod)
				for _, handler := range w.handlers {
					handler.AddResource(podRes)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if pod, ok := newObj.(*corev1.Pod); ok {
				podRes := createResourceFromPod(pod)
				for _, handler := range w.handlers {
					handler.UpdateResource(podRes)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				podRes := createResourceFromPod(pod)
				for _, handler := range w.handlers {
					handler.DeleteResource(podRes)
				}
			}
		},
	})
}

func createResourceFromPod(pod *corev1.Pod) *resource.Resource {
	name2port := make(map[string]string)
	for _, c := range pod.Spec.Containers {
		for _, p := range c.Ports {
			name2port[p.Name] = strconv.Itoa(int(p.ContainerPort))
		}
	}
	return &resource.Resource{
		ResUID:     resource.ResUID(pod.ObjectMeta.UID),
		ResType:    resource.PodType,
		ResVersion: resource.ResVersion(pod.ObjectMeta.ResourceVersion),
		Name:       pod.Name,
		Relations:  getOwnerRef(pod),
		StringAttr: map[resource.AttrKey]string{
			resource.NamespaceAttr:    pod.Namespace,
			resource.ContainerIDsAttr: getContainerIDs(pod),
			resource.PodIP:            pod.Status.PodIP,
			resource.PodPhase:         string(pod.Status.Phase),
			resource.PodHostName:      pod.Spec.NodeName,
			resource.PodHostIP:        pod.Status.HostIP,
		},
		Int64Attr: map[resource.AttrKey]int64{
			resource.PodHostNetwork: getIntForBoolAttr(pod.Spec.HostNetwork),
		},
		ExtraAttr: map[resource.AttrKey]map[string]string{
			resource.PodLabelsAttr: pod.Labels,
			resource.Name2Port:     name2port,
		},
	}
}

func getIntForBoolAttr(val bool) int64 {
	if val {
		return 1
	}
	return 0
}

func getContainerIDs(pod *corev1.Pod) string {
	var str strings.Builder
	var hasBefore bool
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if hasBefore {
			str.WriteByte(',')
		} else {
			hasBefore = true
		}
		str.WriteString(cutContainerId(containerStatus.ContainerID))
	}
	return str.String()
}

func cutContainerId(containerIdStatus string) string {
	containerIdStatus, _ = strings.CutPrefix(containerIdStatus, "containerd://")
	containerIdStatus, _ = strings.CutPrefix(containerIdStatus, "docker://")
	if len(containerIdStatus) > 12 {
		containerIdStatus = containerIdStatus[:12]
	}
	return containerIdStatus
}

func getOwnerRef(pod *corev1.Pod) []resource.Relation {
	var ownerRef []resource.Relation = make([]resource.Relation, 0, len(pod.OwnerReferences))
	for _, owner := range pod.OwnerReferences {
		var relation = resource.Relation{
			ResUID: resource.ResUID(owner.UID),
			ReType: resource.R_OWNER,
			StringAttr: map[resource.AttrKey]string{
				resource.OwnerName: owner.Name,
				resource.OwnerType: string(owner.Kind),
			},
		}
		ownerRef = append(ownerRef, relation)
	}
	return ownerRef
}
