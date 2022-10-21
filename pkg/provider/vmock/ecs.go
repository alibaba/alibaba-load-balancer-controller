package vmock

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"k8s.io/alibaba-load-balancer-controller/pkg/model"
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
)

func NewMockECS(
	auth *base.ClientMgr,
) *MockECS {
	return &MockECS{auth: auth}
}

type MockECS struct {
	auth *base.ClientMgr
}

var _ prvd.IInstance = &MockECS{}

func (d *MockECS) GetInstanceByIp(ip, region, vpc string) ([]ecs.Instance, error) {
	return nil, nil
}
func (d *MockECS) DescribeNetworkInterfaces(vpcId string, ips []string, ipVersionType model.AddressIPVersionType) (map[string]string, error) {
	eniids := make(map[string]string)
	for _, ip := range ips {
		eniids[ip] = "eni-id"
	}
	return eniids, nil
}
