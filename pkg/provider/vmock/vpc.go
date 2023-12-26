package vmock

import (
	"context"

	servicesvpc "github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"

	// "k8s.io/alibaba-load-balancer-controller/pkg/model"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/vpc"
)

func NewMockVPC(
	auth *base.ClientMgr,
) *MockVPC {
	return &MockVPC{auth: auth}
}

type MockVPC struct {
	auth *base.ClientMgr
}

// func (m *MockVPC) CreateRoute(ctx context.Context, table string, provideID string, destinationCIDR string) (*model.Route, error) {
// 	panic("implement me")
// }

func (m *MockVPC) DeleteRoute(ctx context.Context, table, provideID, destinationCIDR string) error {
	panic("implement me")
}

// func (m *MockVPC) ListRoute(ctx context.Context, table string) ([]*model.Route, error) {
// 	panic("implement me")
// }

// func (m *MockVPC) FindRoute(ctx context.Context, table, pvid, cidr string) (*model.Route, error) {
// 	panic("implement me")
// }

func (m *MockVPC) ListRouteTables(ctx context.Context, vpcID string) ([]string, error) {
	switch vpcID {
	case "vpc-no-route-table":
		return []string{}, nil
	case "vpc-single-route-table":
		return []string{"route-table-1"}, nil
	case "vpc-multi-route-table":
		return []string{"route-table-1", "route-table-2"}, nil
	}
	return []string{}, nil
}

func (m *MockVPC) DescribeEipAddresses(ctx context.Context, instanceType string, instanceId string) ([]string, error) {
	if instanceType == string(vpc.SlbInstance) {
		return []string{"172.16.0.10"}, nil
	}
	return nil, nil
}

func (m *MockVPC) DescribeVSwitches(ctx context.Context, vpcID string) ([]servicesvpc.VSwitch, error) {
	panic("implement me")
}
