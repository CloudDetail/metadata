package export_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CloudDetail/metadata/configs"
	"github.com/CloudDetail/metadata/export"
	"github.com/CloudDetail/metadata/model/resource"
	"github.com/CloudDetail/metadata/source"
	"github.com/stretchr/testify/assert"
)

func TestPushToMetaSourceAndClientRestart(t *testing.T) {
	source2, err := testMetaSource2() // listen 3428
	assert.NoError(t, err)
	source2.Run()
	assert.NoError(t, err)

	// source1, err := testMetaSource1() // listen 3427
	// assert.NoError(t, err)
	// source1.Run()
	// assert.NoError(t, err)
	// source1 with push event to source2

	podList := resource.NewResources(resource.PodType, []*resource.Resource{})
	podList.SetClusterID("TEST_CLUSTER")
	httpExporter := export.NewHTTPExporter("http://localhost:3428")
	podList.SetExporter(httpExporter)
	for i := 0; i < 100; i++ {
		event := testPodEvent(i)
		for _, res := range event.Res {
			podList.AddResource(res)
		}
	}

	// wait for 5 second
	time.Sleep(1 * time.Second)
	for i := 100; i < 200; i++ {
		event := testPodEvent(i)
		for _, res := range event.Res {
			podList.AddResource(res)
		}
	}
	time.Sleep(3 * time.Second)

	// check data in source1
	checkByQuery(t, 200)

	// close exporter1
	httpExporter.Stop()

	podList2 := resource.NewResources(resource.PodType, []*resource.Resource{})
	podList2.SetClusterID("TEST_CLUSTER")
	httpExporter2 := export.NewHTTPExporter("http://localhost:3428")
	podList2.SetExporter(httpExporter2)
	for i := 200; i < 300; i++ {
		event := testPodEvent(i)
		for _, res := range event.Res {
			podList2.AddResource(res)
		}
	}

	time.Sleep(5 * time.Second)
	checkByQuery(t, 100)
}

func TestPushToMetaSourceAndServerRestart(t *testing.T) {
	source2, err := testMetaSource2() // listen 3428
	assert.NoError(t, err)
	source2.Run()
	assert.NoError(t, err)

	// source1, err := testMetaSource1() // listen 3427
	// assert.NoError(t, err)
	// source1.Run()
	// assert.NoError(t, err)
	// source1 with push event to source2

	podList := resource.NewResources(resource.PodType, []*resource.Resource{})
	podList.SetClusterID("TEST_CLUSTER")
	httpExporter := export.NewHTTPExporter("http://localhost:3428")
	podList.SetExporter(httpExporter)
	for i := 0; i < 100; i++ {
		event := testPodEvent(i)
		for _, res := range event.Res {
			podList.AddResource(res)
		}
	}

	// wait for 5 second
	time.Sleep(1 * time.Second)
	for i := 100; i < 200; i++ {
		event := testPodEvent(i)
		for _, res := range event.Res {
			podList.AddResource(res)
		}
	}
	time.Sleep(3 * time.Second)

	// check data in source1
	checkByQuery(t, 200)

	// reset data
	source2.Stop()
	source2, err = testMetaSource2() // listen 3428
	assert.NoError(t, err)
	source2.Run()
	assert.NoError(t, err)

	for i := 200; i < 300; i++ {
		event := testPodEvent(i)
		for _, res := range event.Res {
			podList.AddResource(res)
		}
	}

	time.Sleep(5 * time.Second)
	checkByQuery(t, 300)
}

func checkByQuery(t *testing.T, exceptedCount int) {
	reqBody := "{\"ResType\":1,\"ListAll\":true}"
	resp, err := http.Post("http://localhost:3428/query", "application/json", strings.NewReader(reqBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	resMap := make(map[string]any)
	err = json.Unmarshal(data, &resMap)
	assert.NoError(t, err)

	objS := resMap["Object"]
	assert.NotNil(t, objS)

	objList := objS.([]any)
	assert.Equal(t, exceptedCount, len(objList))
}

func testMetaSource1() (source.MetaSource, error) {
	metaSource := source.CreateMetaSourceFromConfig(&configs.MetaSourceConfig{
		AcceptEventSource: &configs.AcceptEventSourceConfig{
			AcceptEventPort: 3427,
		},
		Querier: &configs.QuerierConfig{
			QueryServerPort: 3427,
			IsSingleCluster: false,
		},
		Exporter: &configs.ExporterConfig{
			RemoteWriteAddr: "http://localhost:3428",
		},
	})

	return metaSource, nil
}

func testMetaSource2() (source.MetaSource, error) {
	metaSource := source.CreateMetaSourceFromConfig(&configs.MetaSourceConfig{
		HttpServer: &configs.HTTPServerConfig{
			Port: 3428,
		},
		AcceptEventSource: &configs.AcceptEventSourceConfig{
			EnableAcceptServer: true,
		},
		Querier: &configs.QuerierConfig{
			EnableQueryServer: true,
			IsSingleCluster:   false,
		},
	})

	return metaSource, nil
}
