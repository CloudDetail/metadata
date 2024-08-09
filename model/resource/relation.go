package resource

type RelationType int

const (
	R_OWNER    RelationType = 0x0001
	R_ENDPOINT RelationType = 0x0003
)

type Relation struct {
	ResUID ResUID
	ReType RelationType

	StringAttr map[AttrKey]string `json:"strAttrMap"`
}

type RelationRef struct {
	Relation
	Res *Resource
}
