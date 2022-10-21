package vpc

import (
	"context"

	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/util"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
)

type AssociatedInstanceType string

const SlbInstance = AssociatedInstanceType("SlbInstance")

func NewVPCProvider(
	auth *base.ClientMgr,
) *VPCProvider {
	return &VPCProvider{auth: auth}
}

var _ prvd.IVPC = &VPCProvider{}

type VPCProvider struct {
	auth   *base.ClientMgr
	region string
}

// DescribeVSwitches used for e2etest
func (r *VPCProvider) DescribeVSwitches(ctx context.Context, vpcID string) ([]vpc.VSwitch, error) {
	req := vpc.CreateDescribeVSwitchesRequest()
	req.VpcId = vpcID
	var vSwitches []vpc.VSwitch
	next := &util.Pagination{
		PageNumber: 1,
		PageSize:   10,
	}
	for {
		req.PageSize = requests.NewInteger(next.PageSize)
		req.PageNumber = requests.NewInteger(next.PageNumber)
		resp, err := r.auth.VPC.DescribeVSwitches(req)
		if err != nil {
			return nil, err
		}
		vSwitches = append(vSwitches, resp.VSwitches.VSwitch...)
		pageResult := &util.PaginationResult{
			PageNumber: resp.PageNumber,
			PageSize:   resp.PageSize,
			TotalCount: resp.TotalCount,
		}
		next = pageResult.NextPage()
		if next == nil {
			break
		}
	}
	return vSwitches, nil
}
