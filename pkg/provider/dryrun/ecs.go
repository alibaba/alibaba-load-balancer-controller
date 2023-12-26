package dryrun

import (
	"context"

	sdkecs "github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"k8s.io/alibaba-load-balancer-controller/pkg/model"
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/ecs"
)

func NewDryRunECS(
	auth *base.ClientMgr,
	ecs *ecs.ECSProvider,
) *DryRunECS {
	return &DryRunECS{auth: auth, ecs: ecs}
}

type DryRunECS struct {
	auth *base.ClientMgr
	ecs  *ecs.ECSProvider
}

var _ prvd.IInstance = &DryRunECS{}

func (d *DryRunECS) ListInstances(ctx context.Context, ids []string) (map[string]*prvd.NodeAttribute, error) {
	return d.ecs.ListInstances(ctx, ids)
}

func (d *DryRunECS) GetInstancesByIP(ctx context.Context, ips []string) (*prvd.NodeAttribute, error) {
	return d.ecs.GetInstancesByIP(ctx, ips)
}

func (d *DryRunECS) DescribeNetworkInterfaces(vpcId string, ips []string, ipVersionType model.AddressIPVersionType) (map[string]string, error) {
	return d.ecs.DescribeNetworkInterfaces(vpcId, ips, ipVersionType)
}

func (d *DryRunECS) GetInstanceByIp(ip, region, vpc string) ([]sdkecs.Instance, error) {
	return nil, nil
}
