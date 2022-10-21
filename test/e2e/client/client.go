package client

import (
	"fmt"
	"strings"

	ctrlCfg "k8s.io/alibaba-load-balancer-controller/pkg/config"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/options"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	runtime "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type E2EClient struct {
	CloudClient   *alibaba.AlibabaCloud
	KubeClient    *KubeClient
	DynamicClient dynamic.Interface
	RuntimeClient runtime.Client
}

func NewClient() (*E2EClient, error) {
	ctrlCfg.ControllerCFG.CloudConfigPath = options.TestConfig.CloudConfig

	fmt.Println("%#v", options.TestConfig)
	// alb测试账号没有ack资源权限，跳过ackClient

	newCC := alibaba.NewAlibabaCloud().(*alibaba.AlibabaCloud)

	cfg := config.GetConfigOrDie()
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("new client : %s", err.Error()))
	}
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("new dynamic client : %s", err.Error()))
	}
	runtimeClient, err := runtime.New(cfg, runtime.Options{})
	if err != nil {
		panic(fmt.Sprintf("new runtime client error: %s", err.Error()))
	}

	return &E2EClient{
		CloudClient:   newCC,
		KubeClient:    NewKubeClient(kubeClient),
		DynamicClient: dynamicClient,
		RuntimeClient: runtimeClient,
	}, nil
}

func InitCloudConfig(client *ACKClient) error {
	ack, err := client.DescribeClusterDetail(options.TestConfig.ClusterId)
	if err != nil {
		return err
	}
	if ctrlCfg.CloudCFG.Global.Region == "" {
		ctrlCfg.CloudCFG.Global.Region = *ack.RegionId
	}
	if ctrlCfg.CloudCFG.Global.ClusterID == "" {
		ctrlCfg.CloudCFG.Global.ClusterID = *ack.ClusterId
	}
	if ctrlCfg.CloudCFG.Global.VswitchID == "" {
		vswitchIds := strings.Split(*ack.VswitchId, ",")
		if len(vswitchIds) > 1 {
			ctrlCfg.CloudCFG.Global.VswitchID = vswitchIds[0]
		} else {
			ctrlCfg.CloudCFG.Global.VswitchID = *ack.VswitchId
		}
	}
	if ctrlCfg.CloudCFG.Global.VpcID == "" {
		ctrlCfg.CloudCFG.Global.VpcID = *ack.VpcId
	}

	return nil
}
