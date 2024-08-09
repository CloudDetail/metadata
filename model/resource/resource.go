package resource

type ResVersion string
type ResUID string

type Resource struct {
	ResUID
	ResType

	// Last updated timestamp
	ResVersion

	Name string `json:"name"`

	Relations []Relation `json:"relations"`

	StringAttr map[AttrKey]string `json:"strAttrMap"`
	Int64Attr  map[AttrKey]int64  `json:"int64AttrMap"`

	// Labels for Pod
	// Selector for services
	ExtraAttr map[AttrKey]map[string]string `json:"extraInfo"`
}
