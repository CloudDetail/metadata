package resource

type ResOperation int

const (
	AddOP    ResOperation = 0
	UpdateOP ResOperation = 1
	DeleteOP ResOperation = 2

	ResetOP ResOperation = 3
)

type ResourceEvent struct {
	// 事件来源的集群ID
	ClusterID string

	Res          []*Resource
	ResourceType ResType
	Operation    ResOperation
}

type SyncRequest struct {
	Events []*ResourceEvent
	// 上次更新时间
	LastCheckPoint *CheckPoint
	// 本次更新结束时间
	CheckPoint *CheckPoint
}

// IsSyncCheck 是否是同步检查信号
// CheckPoint和Events都为nil, 表示不做更新, 仅检查LastCheckPoint状态
func (r *SyncRequest) IsSyncCheck() bool {
	return r.CheckPoint == nil && r.Events == nil && r.LastCheckPoint != nil
}

// IsHealthCheck 是否是初始化信号
// LastCheckPoint为nil,表示未曾提交过数据,此时不会提供其他数据
func (r *SyncRequest) IsHealthCheck() bool {
	return r.LastCheckPoint == nil && r.CheckPoint == nil
}

func (r *SyncRequest) IsInitRequest() bool {
	return r.LastCheckPoint == nil && r.CheckPoint != nil
}

type CheckPoint struct {
	AgentIndex int64
	Timestamp  int64
	EventIndex int
}

func (p *CheckPoint) Equals(cp *CheckPoint) bool {
	if p == nil || cp == nil {
		return false
	}
	return p.Timestamp == cp.Timestamp && p.EventIndex == cp.EventIndex
}

type SyncResponse struct {
	LastCheckPoint *CheckPoint
	IsStopPush     bool
	IsInit         bool
	IsAccepted     bool
}
