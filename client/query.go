package client

import "github.com/CloudDetail/metadata/model/resource"

type ResQueryRequest struct {
	*resource.ResType
	*resource.ResUID

	K8s *struct {
		Name      string
		Namespace string
	}

	ListAll bool
}
