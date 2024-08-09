package apiserver

import (
	"context"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type IWatcher interface {
	Init(
		ctx context.Context,
		client *kubernetes.Clientset,
		factory informers.SharedInformerFactory,
		namespace string,
		handlersMap ResourceHandlersMap,
	)
	Run()
}
