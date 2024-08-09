package resource

type AttrKey int

// K8sMetadata
const (
	NamespaceAttr AttrKey = 0x0000

	// K8sPod
	ContainerIDsAttr AttrKey = 0x0010 // string containerID1,containerID2,...
	PodLabelsAttr    AttrKey = 0x0011 // extra map[string]string
	PodIP            AttrKey = 0x0012 // string
	PodPhase         AttrKey = 0x0013 // string Pending / Running
	PodHostName      AttrKey = 0x0014 // string
	PodHostIP        AttrKey = 0x0015 // string
	PodHostNetwork   AttrKey = 0x0016 // bool

	// K8sService
	ServiceSelectorsAttr     AttrKey = 0x0020 // extra map[string]string
	ServiceIP                AttrKey = 0x0021 // string
	ServiceEndpoints         AttrKey = 0x0022 // string ip1,ip2,...
	ServicePorts2TargetPorts AttrKey = 0x0023 // string name:port-targetPort-nodePort,name2:port-targetPort-nodePort

	// Node
	NodeInternalIP AttrKey = 0x0030
	NodeExternalIP AttrKey = 0x0031
	NodeHostName   AttrKey = 0x0032

	// OwnerAttribute
	OwnerName AttrKey = 0x0111
	OwnerType AttrKey = 0x0112
)
