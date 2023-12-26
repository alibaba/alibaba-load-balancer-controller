package vmock

import (
	"context"

	sdkecs "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"k8s.io/alibaba-load-balancer-controller/pkg/model"
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	v1 "k8s.io/api/core/v1"
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

const (
	ZoneID       = "cn-hangzhou-a"
	RegionID     = "cn-hangzhou"
	InstanceIP   = "192.0.168.68"
	InstanceType = "ecs.c6.xlarge"
)

func (d *MockECS) ListInstances(ctx context.Context, ids []string) (map[string]*prvd.NodeAttribute, error) {
	mins := make(map[string]*prvd.NodeAttribute)
	for _, id := range ids {
		mins[id] = &prvd.NodeAttribute{
			InstanceID:   id,
			InstanceType: InstanceType,
			Addresses: []v1.NodeAddress{
				{
					Type:    v1.NodeInternalIP,
					Address: InstanceIP,
				},
			},
			Zone:   ZoneID,
			Region: RegionID,
		}
	}
	return mins, nil
}

func (d *MockECS) GetInstancesByIP(ctx context.Context, ips []string) (*prvd.NodeAttribute, error) {
	return nil, nil
}

func (d *MockECS) GetInstanceByIp(ip, region, vpc string) ([]sdkecs.Instance, error) {
	return nil, nil
}

func (d *MockECS) DescribeNetworkInterfaces(vpcId string, ips []string, ipVersionType model.AddressIPVersionType) (map[string]string, error) {
	eniids := make(map[string]string)
	for _, ip := range ips {
		eniids[ip] = "eni-id"
	}
	return eniids, nil
}
