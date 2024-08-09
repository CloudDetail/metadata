package resource

type ResType int

const (
	PodType     ResType = 0x0001
	ServiceType ResType = 0x0002
	NodeType    ResType = 0x0003
)
