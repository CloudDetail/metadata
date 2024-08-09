package apiserver

import (
	"context"

	"github.com/CloudDetail/metadata/model/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func init() {
	addWatcher(resource.NodeType, &NodeWatcher{})
}

type NodeWatcher struct {
	ctx    context.Context
	client *kubernetes.Clientset

	factory informers.SharedInformerFactory

	handlers []resource.ResHandler
}

func (w *NodeWatcher) Init(ctx context.Context, client *kubernetes.Clientset, factory informers.SharedInformerFactory, namespace string, handlersMap ResourceHandlersMap) {
	w.ctx = ctx
	w.client = client
	w.factory = factory
	w.handlers = handlersMap[resource.NodeType]
}

func (w *NodeWatcher) Run() {
	informer := w.factory.Core().V1().Nodes().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if node, ok := obj.(*corev1.Node); ok {
				res := createResourceFromNode(node)
				for _, handler := range w.handlers {
					handler.AddResource(res)
				}
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if node, ok := newObj.(*corev1.Node); ok {
				res := createResourceFromNode(node)
				for _, handler := range w.handlers {
					handler.UpdateResource(res)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if node, ok := obj.(*corev1.Node); ok {
				res := createResourceFromNode(node)
				for _, handler := range w.handlers {
					handler.DeleteResource(res)
				}
			}
		},
	})
}

func createResourceFromNode(node *corev1.Node) *resource.Resource {
	res := &resource.Resource{
		ResUID:     resource.ResUID(node.UID),
		ResType:    resource.NodeType,
		ResVersion: resource.ResVersion(node.ResourceVersion),
		Name:       node.Name,
		Relations:  []resource.Relation{},
		StringAttr: map[resource.AttrKey]string{
			// resource.NodeInternalIP:
		},
		Int64Attr: map[resource.AttrKey]int64{},
		ExtraAttr: map[resource.AttrKey]map[string]string{},
	}
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			res.StringAttr[resource.NodeInternalIP] = address.Address
		} else if address.Type == corev1.NodeExternalIP {
			res.StringAttr[resource.NodeExternalIP] = address.Address
		} else if address.Type == corev1.NodeHostName {
			res.StringAttr[resource.NodeHostName] = address.Address
		}
	}

	return res
}
