package dryrun

import (
	"context"

	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/vpc"

	servicesvpc "github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
)

func NewDryRunVPC(
	auth *base.ClientMgr,
	vpc *vpc.VPCProvider,
) *DryRunVPC {
	return &DryRunVPC{auth: auth, vpc: vpc}
}

type DryRunVPC struct {
	auth *base.ClientMgr
	vpc  *vpc.VPCProvider
}

func (m *DryRunVPC) DescribeVSwitches(ctx context.Context, vpcID string) ([]servicesvpc.VSwitch, error) {
	return m.vpc.DescribeVSwitches(ctx, vpcID)
}
