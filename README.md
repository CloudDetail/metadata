# metadata

一个用于采集/传输/查询元信息模块;

## 创建MetaSource实例

```golang

func main() {
    cfg := createDefaultConfig()
    ms := source.CreateMetaSourceFromConfig(cfg)
    err := ms.Run()
    if err != nil {
        log.Panic(err)
    }
}

func createDefaultConfig() *configs.MetaSourceConfig {
    return &configs.MetaSourceConfig{
        HttpServer: &configs.HTTPServerConfig{
            Port: 8080,
        },
        KubeSource: &configs.KubeSourceConfig{
            KubeAuthType:      "serviceAccount",
            IsEndpointsNeeded: true,
        },
        Querier: &configs.QuerierConfig{
            EnableQueryServer: true,
            IsSingleCluster:   true,
        },
        Exporter: &configs.ExporterConfig{
            EnableFetchServer: true,
        },
    }
}
```

## 数据源来源

数据源目前可以来自K8sAPIServer或者其他MetaSource实例
不同MetaSource实例之间支持通过HTTP请求以Pull/Push方式传输数据.
