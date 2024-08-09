package metasource

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/CloudDetail/metadata/model/resource"
)

func (r *MetaSource) HandlePushedEvent(w http.ResponseWriter, req *http.Request) {
	var syncReq resource.SyncRequest

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("parse push event failed: failed to read request body: err: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if err := json.Unmarshal(body, &syncReq); err != nil {
		log.Printf("parse push event failed:  failed to unmarshal request body: err: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp := resource.SyncResponse{
		IsAccepted: true,
	}

	if syncReq.IsHealthCheck() {
		// 为初始化探针提供AgentIndex编号
		resp.LastCheckPoint = &resource.CheckPoint{
			AgentIndex: r.AgentCounter.Add(1),
			EventIndex: 0,
		}
	} else if syncReq.IsSyncCheck() {
		// 同步性检查时,不传递事件,只传递信息编号
		checkPoint, find := r.AgentLastCheckPoint.Get(syncReq.LastCheckPoint.AgentIndex)
		if !find || checkPoint.EventIndex != syncReq.LastCheckPoint.EventIndex {
			// 重新初始化
			resp.IsInit = true
		}
	} else {
		resp.LastCheckPoint, resp.IsInit = r.handlerSyncRequest(&syncReq)
		r.AgentLastCheckPoint.Add(syncReq.CheckPoint.AgentIndex, syncReq.CheckPoint)
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("sync with meta-agent failed: failed to marshal response body: err: %v", err)
		return
	}
	w.Write(jsonResp)
}

func (r *MetaSource) handlerSyncRequest(syncReq *resource.SyncRequest) (cp *resource.CheckPoint, isInit bool) {
	for _, event := range syncReq.Events {
		handlerMap, find := r.ClusterMaps.Load(event.ClusterID)
		if !find {
			handlerMap = r.initClusterHandlerMap(event.ClusterID)
			r.ClusterMaps.Store(event.ClusterID, handlerMap)
		}

		if !find && event.Operation != resource.ResetOP {
			// 未初始化过的cluster,但不是reset事件,直接请求重新发送
			log.Printf("[%s] accept meta event on uninitialized cluster, ask for reset", event.ClusterID)
			return nil, true
		} else if event.Operation == resource.ResetOP {
			log.Printf("[%s] accept meta reset (%d) event", event.ClusterID, event.ResourceType)
		}

		handlerMap.(*ClusterHandlerMap).HandlerEvent(event)
	}

	return syncReq.CheckPoint, false
}
