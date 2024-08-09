package export_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/CloudDetail/metadata/export"
	"github.com/CloudDetail/metadata/model/resource"
	"github.com/CloudDetail/metadata/server"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchFromServer2(t *testing.T) {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/metadata/fetch"}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		testFetchClient(t, &wg, nil, u, resource.PodType)
	}()

	wg.Wait()
}

func TestFetchFromServer(t *testing.T) {
	httpServer := server.NewHTTPServer(fmt.Sprintf(":%d", 8080))

	fetchServer := export.NewFetcherServer()
	httpServer.RegisterHandler("/fetch", fetchServer.FetchWithWS)

	err := httpServer.StartHttpServer()
	assert.NoError(t, err)

	podList := resource.NewResources(resource.PodType, nil)
	podList.SetExporter(fetchServer)
	serviceList := resource.NewResources(resource.ServiceType, nil)
	serviceList.SetExporter(fetchServer)

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/fetch"}

	podData := []*resource.ResourceEvent{}
	for i := 0; i < 100; i++ {
		podData = append(podData, testPodEvent(i))
	}

	serviceData := []*resource.ResourceEvent{}
	for i := 0; i < 100; i++ {
		serviceData = append(serviceData, testServiceEvent(i))
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		testFetchClient(t, &wg, podList, u, resource.PodType)
	}()

	time.Sleep(3 * time.Second)

	go func() {
		for _, event := range podData {
			podList.AddResource(event.Res[0])
		}
	}()

	for _, event := range serviceData {
		serviceList.AddResource(event.Res[0])
	}

	time.Sleep(3 * time.Second)

	go func() {
		testFetchClient(t, &wg, serviceList, u, resource.ServiceType)
	}()

	wg.Wait()
}

func testFetchClient(
	t *testing.T,
	wg *sync.WaitGroup,
	expected *resource.Resources,
	u url.URL,
	resTypes ...resource.ResType,
) {
	defer wg.Done()
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(t, err, "failed to connect with fetchServer")
	defer conn.Close()

	fetchRequest := resource.FetchRequest{
		ResourceTypes: resTypes,
	}

	data, err := json.Marshal(fetchRequest)
	assert.NoError(t, err, "failed to marshal fetch request")
	err = conn.WriteMessage(websocket.BinaryMessage, data)

	assert.NoError(t, err, "failed to write fetch request")

	resList := &resource.Resources{
		ResList:   []*resource.Resource{},
		ResType:   resTypes[0],
		ClusterID: "TEST_CLUSTER",
		Exporter:  export.NonExporter,
	}
	go readAndFillResList(conn, resList)
	<-time.After(5 * time.Second)

	require.Equal(t, len(expected.ResList), len(resList.ResList), "resource count not match")
	for index, res := range expected.ResList {
		assert.Equal(t, *res, *resList.ResList[index], "resource order not match")
	}

}

func readAndFillResList(conn *websocket.Conn, resList resource.ResHandler) {
	for {
		var syncReq resource.SyncRequest
		err := conn.ReadJSON(&syncReq)
		if err != nil {
			return
		}
		for _, event := range syncReq.Events {
			switch event.Operation {
			case resource.AddOP:
				for _, res := range event.Res {
					resList.AddResource(res)
				}
			case resource.ResetOP:
				resList.Reset(event.Res)
			}
		}
	}
}

func testPodEvent(podID int) *resource.ResourceEvent {
	return &resource.ResourceEvent{
		ClusterID: "TEST_CLUSTER",
		Res: []*resource.Resource{
			{
				ResUID:     resource.ResUID(strconv.Itoa(podID)),
				ResType:    resource.PodType,
				ResVersion: resource.ResVersion(strconv.Itoa(podID)),
				Name:       strconv.Itoa(podID),
				Relations:  []resource.Relation{},
				StringAttr: map[resource.AttrKey]string{
					resource.PodIP: "1.1.1.1",
				},
				Int64Attr: map[resource.AttrKey]int64{},
				ExtraAttr: map[resource.AttrKey]map[string]string{
					resource.PodLabelsAttr: {
						"label1": "1123",
					},
				},
			},
		},
		ResourceType: resource.PodType,
		Operation:    resource.AddOP,
	}
}

func testServiceEvent(serviceID int) *resource.ResourceEvent {
	return &resource.ResourceEvent{
		ClusterID: "TEST_CLUSTER",
		Res: []*resource.Resource{
			{
				ResUID:     resource.ResUID(strconv.Itoa(serviceID)),
				ResType:    resource.ServiceType,
				ResVersion: resource.ResVersion(strconv.Itoa(serviceID)),
				Name:       strconv.Itoa(serviceID),
				Relations:  []resource.Relation{},
				StringAttr: map[resource.AttrKey]string{
					resource.ServiceIP: "1.1.1.1",
				},
				Int64Attr: map[resource.AttrKey]int64{},
				ExtraAttr: map[resource.AttrKey]map[string]string{
					resource.ServiceSelectorsAttr: {
						"label1": "1123",
					},
				},
			},
		},
		ResourceType: resource.ServiceType,
		Operation:    resource.AddOP,
	}
}
