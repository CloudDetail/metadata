package metasource

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CloudDetail/metadata/model/resource"

	"github.com/gorilla/websocket"
)

func (r *MetaSource) RunWithFetcher(address string, resTypes ...resource.ResType) error {
	address = strings.TrimPrefix(address, "http://")
	// 移除末尾的/
	address = strings.TrimSuffix(address, "/")
	pathIdx := strings.IndexByte(address, '/')
	var u url.URL
	if pathIdx < 0 {
		u = url.URL{Scheme: "ws", Host: address, Path: "/fetch"}
	} else {
		u = url.URL{Scheme: "ws", Host: address[:pathIdx], Path: address[pathIdx:] + "/fetch"}
	}

	for {
		err := r.fetchFrom(u, resTypes...)
		if err != nil {
			log.Printf("failed to fetch from source[%s], retry after 30s", address)
			time.Sleep(30 * time.Second)
		} else {
			return nil
		}
	}
}

func (r *MetaSource) fetchFrom(u url.URL, resTypes ...resource.ResType) error {
	var fetchHeader = http.Header{"X-Data-Flow": {"meta-fetch"}}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), fetchHeader)
	if err != nil {
		return err
	}
	defer conn.Close()
	r.stop = conn.Close

	fetchRequest := resource.FetchRequest{
		ResourceTypes: resTypes,
	}

	data, err := json.Marshal(fetchRequest)
	if err != nil {
		return err
	}
	err = conn.WriteMessage(websocket.BinaryMessage, data)

	if err != nil {
		return err
	}

	log.Printf("success fetch from source[%s], keep reading", u.String())
	for {
		_, received, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var syncReq resource.SyncRequest
		err = json.Unmarshal(received, &syncReq)
		if err != nil {
			return err
		}

		for _, event := range syncReq.Events {
			handlerMap, find := r.ClusterMaps.Load(event.ClusterID)
			if !find {
				handlerMap = r.initClusterHandlerMap(event.ClusterID)
				r.ClusterMaps.Store(event.ClusterID, handlerMap)
			}

			if !find && event.Operation != resource.ResetOP {
				// 未初始化过的cluster,但不是reset事件,直接请求重新发送
				log.Printf("[%s] accept meta event on uninitialized cluster, ask for reset", event.ClusterID)
			} else if event.Operation == resource.ResetOP {
				log.Printf("[%s] accept meta reset (%d) event", event.ClusterID, event.ResourceType)
			}

			handlerMap.(*ClusterHandlerMap).HandlerEvent(event)
		}
	}
}
