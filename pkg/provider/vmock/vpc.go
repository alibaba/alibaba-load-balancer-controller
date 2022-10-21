package vmock

import (
	"context"

	servicesvpc "github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
)

func NewMockVPC(
	auth *base.ClientMgr,
) *MockVPC {
	return &MockVPC{auth: auth}
}

type MockVPC struct {
	auth *base.ClientMgr
}

func (m *MockVPC) DescribeVSwitches(ctx context.Context, vpcID string) ([]servicesvpc.VSwitch, error) {
	panic("implement me")
}
