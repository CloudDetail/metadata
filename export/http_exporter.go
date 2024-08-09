package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/CloudDetail/metadata/model/resource"
)

var _ resource.Exporter = &HTTPExporter{}

const PushPath = "/push"

type HTTPExporter struct {
	RemoteAddr string
	// 服务端出现严重错误时
	// 控制客户端停止发送
	IsStopPush       bool
	IsServerNotReady bool

	client *http.Client

	messageCounter int
	LastCheckPoint *resource.CheckPoint

	// 初始化时统计全部资源
	resourcesRef []*resource.Resources

	eventChan chan *resource.ResourceEvent
	batch     []*resource.ResourceEvent
	ticker    *time.Ticker

	AgentIndex int64

	failedTime int
}

func NewHTTPExporter(remoteAddr string) *HTTPExporter {
	if !strings.HasPrefix(remoteAddr, "http") {
		remoteAddr = "http://" + remoteAddr
	}
	remoteAddr = strings.TrimSuffix(remoteAddr, "/") + PushPath

	exporter := &HTTPExporter{
		RemoteAddr:       remoteAddr,
		IsStopPush:       false,
		IsServerNotReady: true,
		client:           createHTTPClient(),
		messageCounter:   0,
		LastCheckPoint:   nil,
		resourcesRef:     []*resource.Resources{},
		eventChan:        make(chan *resource.ResourceEvent),
		batch:            []*resource.ResourceEvent{},
	}

	exporter.ticker = time.NewTicker(3 * time.Second)

	go exporter.KeepPushingEvent()
	return exporter
}

func (h *HTTPExporter) CheckIsServerReadyAndInit() bool {
	// 清空未初始化之前batch和channel中的数据
	for len(h.eventChan) > 0 {
		<-h.eventChan
	}
	h.batch = []*resource.ResourceEvent{}
	if !h.checkHealth() {
		return false
	}
	h.failedTime = 0

	h.messageCounter++
	newCheckPoint := &resource.CheckPoint{
		AgentIndex: h.AgentIndex,
		Timestamp:  time.Now().Unix(),
		EventIndex: h.messageCounter,
	}
	resp, err := h.pushInitEvent(newCheckPoint)
	if err != nil {
		return false
	}

	h.LastCheckPoint = resp.LastCheckPoint
	h.IsServerNotReady = false
	log.Printf("meta-server [%s] is ready for pushing event", h.RemoteAddr)
	return true
}

func (h *HTTPExporter) KeepPushingEvent() {
	isReady := h.CheckIsServerReadyAndInit()
	if !isReady {
		log.Printf("meta-server [%s] is not ready, stop pushing event", h.RemoteAddr)
	}
	for {
		select {
		case <-h.ticker.C:
			if h.IsServerNotReady {
				h.CheckIsServerReadyAndInit()
				continue
			}

			if len(h.batch) == 0 {
				if !h.syncCheck() {
					log.Printf("remote is not sync with agent, prepare to init again")
					h.IsServerNotReady = false
				}
				continue
			}

			h.messageCounter++
			nowCP := &resource.CheckPoint{
				AgentIndex: h.AgentIndex,
				Timestamp:  time.Now().Unix(),
				EventIndex: h.messageCounter,
			}

			resp, err := h.pushEvent(h.batch, h.LastCheckPoint, nowCP)
			if err != nil {
				log.Printf("meta-server [%s] is not ready, prepare to init again, err:%v", h.RemoteAddr, err)
				h.IsServerNotReady = true
				h.CheckIsServerReadyAndInit()
				continue
			}

			if resp.IsInit {
				log.Printf("remote is not sync with agent, prepare to init again")
				h.IsServerNotReady = true
				h.CheckIsServerReadyAndInit()
				continue
			}

			h.LastCheckPoint = nowCP

			// 清空batch
			h.batch = []*resource.ResourceEvent{}
		case event, ok := <-h.eventChan:
			if !ok {
				return
			}
			h.batch = append(h.batch, event)
		}
	}
}

// syncCheck同步检查
func (h *HTTPExporter) syncCheck() bool {
	// 检查服务端是否是最新
	resp, err := h.pushEvent(nil, h.LastCheckPoint, nil)
	if err != nil {
		log.Printf("meta-server [%s] is not ready, stop pushing event, err:%v", h.RemoteAddr, err)
		return false
	}

	return !resp.IsInit
}

func (h *HTTPExporter) Stop() {
	h.IsStopPush = true
	close(h.eventChan)
}

func (h *HTTPExporter) checkHealth() bool {
	// 健康检查时不传递任何数据
	resp, err := h.pushEvent(nil, nil, nil)
	if err != nil {
		h.failedTime++
		if h.failedTime%10 == 1 {
			log.Printf("meta-server is not healthy, err: %v", err)
		}
		return false
	}

	if resp.IsAccepted {
		h.AgentIndex = resp.LastCheckPoint.AgentIndex
	}

	return resp.IsAccepted
}

func (h *HTTPExporter) pushInitEvent(newCheckPoint *resource.CheckPoint) (*resource.SyncResponse, error) {
	log.Printf("send init event to reset remote meta [%s]", h.RemoteAddr)
	resetEvents := make([]*resource.ResourceEvent, 0, len(h.resourcesRef))
	for _, res := range h.resourcesRef {
		res.ExportMux.RLock()
		resetEvents = append(resetEvents, &resource.ResourceEvent{
			ClusterID:    res.ClusterID,
			Res:          res.ResList,
			ResourceType: res.ResType,
			Operation:    resource.ResetOP,
		})
		res.ExportMux.RUnlock()
	}
	return h.pushEvent(resetEvents, nil, newCheckPoint)
}

func (h *HTTPExporter) ExportResourceEvents(events *resource.ResourceEvent) {
	if h.IsStopPush || h.IsServerNotReady {
		return
	}

	idleMax := 10 * time.Second
	idleTimeout := time.NewTimer(idleMax)
	defer idleTimeout.Stop()
	select {
	case h.eventChan <- events:
	case <-idleTimeout.C:
		// 放弃插入
		log.Printf("exporter is not ready, prepare to init again")
		h.IsServerNotReady = true
		return
	}
}

func (h *HTTPExporter) pushEvent(events []*resource.ResourceEvent, lastCheckPoint *resource.CheckPoint, newCheckPoint *resource.CheckPoint) (*resource.SyncResponse, error) {
	syncReq := &resource.SyncRequest{
		Events:         events,
		LastCheckPoint: lastCheckPoint,
		CheckPoint:     newCheckPoint,
	}

	body, err := json.Marshal(syncReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, h.RemoteAddr, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Data-Flow", "meta-push")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 400 {
		return nil, fmt.Errorf("server is not ready: stateCode %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response body: err: %v", err)
		return nil, err
	}

	var response resource.SyncResponse
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		log.Printf("failed to read response body: err: %v", err)
		return nil, err
	}

	h.IsStopPush = response.IsStopPush
	return &response, nil
}

func (h *HTTPExporter) SetupResourcesRef(resources *resource.Resources) {
	h.resourcesRef = append(h.resourcesRef, resources)

	if h.IsServerNotReady {
		log.Printf("setup resource [%s](%d), ignore init event since http remote is not ready", resources.ClusterID, resources.ResType)
	} else {
		log.Printf("setup resource [%s](%d), send init event to remote metasource", resources.ClusterID, resources.ResType)
		// 生成资源对应的Init事件
		initEvent := &resource.ResourceEvent{
			ClusterID:    resources.ClusterID,
			Res:          resources.ResList,
			ResourceType: resources.ResType,
			Operation:    resource.ResetOP,
		}

		idleMax := 10 * time.Second
		idleTimeout := time.NewTimer(idleMax)
		defer idleTimeout.Stop()
		select {
		case h.eventChan <- initEvent:
		case <-idleTimeout.C:
			// 放弃插入
			log.Printf("exporter is not ready, prepare to init again")
			h.IsServerNotReady = true
			return
		}
	}
}

func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}
	return client
}
