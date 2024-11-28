package apiserver

import (
	"context"
	"strconv"

	"github.com/CloudDetail/metadata/model/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func init() {
	addWatcher(resource.ServiceType, &ServiceWatcher{})
}

type ServiceWatcher struct {
	ctx     context.Context
	client  *kubernetes.Clientset
	factory informers.SharedInformerFactory

	handlers  []resource.ResHandler
	namespace string
}

func (w *ServiceWatcher) Init(
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
	w.handlers = handlersMap[resource.ServiceType]
}

func (w *ServiceWatcher) Run() {
	informer := w.factory.Core().V1().Services().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if service, ok := obj.(*corev1.Service); ok {
				serviceRes := w.createResourceFromService(service)
				for _, handler := range w.handlers {
					handler.AddResource(serviceRes)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if service, ok := newObj.(*corev1.Service); ok {
				serviceRes := w.createResourceFromService(service)
				for _, handler := range w.handlers {
					handler.UpdateResource(serviceRes)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if service, ok := obj.(*corev1.Service); ok {
				serviceRes := w.createResourceFromService(service)
				for _, handler := range w.handlers {
					handler.DeleteResource(serviceRes)
				}
			}
		},
	})
}

func (*ServiceWatcher) createResourceFromService(eService *corev1.Service) *resource.Resource {
	svc2target := make(map[string]string)
	for _, port := range eService.Spec.Ports {
		svc2target[strconv.Itoa(int(port.Port))] = port.TargetPort.String()
	}

	res := &resource.Resource{
		ResUID:     resource.ResUID(eService.UID),
		ResType:    resource.ServiceType,
		ResVersion: resource.ResVersion(eService.ObjectMeta.ResourceVersion),
		Name:       eService.Name,
		Relations:  []resource.Relation{},
		StringAttr: map[resource.AttrKey]string{
			resource.NamespaceAttr: eService.Namespace,
			resource.ServiceIP:     eService.Spec.ClusterIP,
		},
		Int64Attr: map[resource.AttrKey]int64{},
		ExtraAttr: map[resource.AttrKey]map[string]string{
			resource.ServiceSelectorsAttr:     eService.Spec.Selector,
			resource.ServicePorts2TargetPorts: svc2target,
		},
	}
	return res
}
