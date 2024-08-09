package export

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CloudDetail/metadata/model/resource"
	"github.com/gorilla/websocket"
)

type FetcherServer struct {
	ctx      context.Context
	upgrader *websocket.Upgrader

	resources []*resource.Resources

	registerFetcher   atomic.Int64
	unRegisterFetcher atomic.Int64

	fetchers sync.Map
}

func NewFetcherServer() *FetcherServer {
	srv := &FetcherServer{
		ctx:       context.Background(),
		upgrader:  &websocket.Upgrader{},
		resources: []*resource.Resources{},
	}
	return srv
}

func (s *FetcherServer) SetupResourcesRef(resources *resource.Resources) {
	s.resources = append(s.resources, resources)
	event := &resource.ResourceEvent{
		ClusterID:    resources.ClusterID,
		Res:          resources.ResList,
		ResourceType: resources.ResType,
		Operation:    resource.ResetOP,
	}

	s.ExportResourceEvents(event)
}

func (s *FetcherServer) ExportResourceEvents(event *resource.ResourceEvent) {
	syncReq := &resource.SyncRequest{
		Events: []*resource.ResourceEvent{event},
	}

	data, err := json.Marshal(syncReq)
	if err != nil {
		return
	}

	idleMax := 10 * time.Second
	idleTimeout := time.NewTimer(idleMax)
	defer idleTimeout.Stop()
	s.fetchers.Range(func(_, value any) bool {
		fetcher := value.(*Fetcher)
		_, find := fetcher.FetchedTypes[event.ResourceType]
		if len(fetcher.FetchedTypes) > 0 && !find {
			return true
		}

		idleTimeout.Reset(idleMax)
		select {
		case fetcher.sendChan <- data:
		case <-idleTimeout.C:
			// 放弃插入,该fetcher已经异常
			s.UnregisterFetcher(fetcher)
		}
		return true
	})
}

func (s *FetcherServer) FetchWithWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// TODO return with error
		log.Printf("Upgrade error: %v\n", err)
		return
	}
	defer conn.Close()
	log.Printf("receive fetch request from %s", conn.RemoteAddr())
	fetcher, err := s.RegisterFetcher(conn)
	defer s.UnregisterFetcher(fetcher)
	log.Printf("add fetcher, fetcher list size: %d", s.registerFetcher.Load()-s.unRegisterFetcher.Load())
	if err != nil {
		log.Printf("Upgrade error: %v\n", err)
		return
	}

	var initRequest = resource.SyncRequest{
		Events: []*resource.ResourceEvent{},
	}
	for _, res := range s.resources {
		if len(fetcher.FetchedTypes) > 0 {
			if _, find := fetcher.FetchedTypes[res.ResType]; !find {
				continue
			}
		}

		res.ExportMux.RLock()
		initRequest.Events = append(initRequest.Events, &resource.ResourceEvent{
			ClusterID:    res.ClusterID,
			Res:          res.ResList,
			ResourceType: res.ResType,
			Operation:    resource.ResetOP,
		})
		res.ExportMux.RUnlock()
	}

	data, err := json.Marshal(initRequest)
	if err != nil {
		return
	}

	err = fetcher.PushInitEvent(data)
	if err != nil {
		return
	}
	fetcher.KeepPush()
}

type FetchResponse struct {
}

func (s *FetcherServer) RegisterFetcher(conn *websocket.Conn) (*Fetcher, error) {
	var request resource.FetchRequest
	err := conn.ReadJSON(&request)
	if err != nil {
		return nil, err
	}

	f := &Fetcher{
		ID:           s.registerFetcher.Add(1),
		ctx:          s.ctx,
		FetchedTypes: fetchedTypesMap(request.ResourceTypes),
		conn:         conn,
		sendChan:     make(chan []byte),
	}
	s.fetchers.Store(f.ID, f)
	return f, nil
}

func fetchedTypesMap(types []resource.ResType) map[resource.ResType]struct{} {
	if len(types) == 0 {
		return nil
	}
	var typeMap = make(map[resource.ResType]struct{}, len(types))
	for _, resType := range types {
		typeMap[resType] = struct{}{}
	}
	return typeMap
}

func (s *FetcherServer) UnregisterFetcher(f *Fetcher) {
	if f.isClosed.Swap(true) {
		return
	}
	s.fetchers.Delete(f.ID)
	log.Printf("unregister fetcher [%s], fetcher list size: %d", f.conn.RemoteAddr().String(), s.registerFetcher.Load()-s.unRegisterFetcher.Add(1))
}

func (s *FetcherServer) Stop() {
	s.ctx.Done()
}

type Fetcher struct {
	ID int64

	isClosed atomic.Bool

	ctx          context.Context
	FetchedTypes map[resource.ResType]struct{}
	// Web Socket
	conn *websocket.Conn

	// Send
	sendChan chan []byte
}

func (f *Fetcher) PushInitEvent(data []byte) error {
	return f.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (f *Fetcher) KeepPush() {
	heartBeatTicker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-heartBeatTicker.C:
			err := f.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				// 心跳检查失败
				log.Printf("fetcher [%s] failed at heart beat", f.conn.RemoteAddr().String())
				return
			}
		case <-f.ctx.Done():
			return
		case data := <-f.sendChan:
			err := f.conn.WriteMessage(websocket.BinaryMessage, data)
			if err != nil {
				return
			}
		}
	}
}
