package ecs

import (
	"encoding/json"
	"fmt"

	"k8s.io/alibaba-load-balancer-controller/pkg/model"
	"k8s.io/klog/v2"

	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/util"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

const (
	MaxNetworkInterfaceNum = 100
)

func NewECSProvider(
	auth *base.ClientMgr,
) *ECSProvider {
	return &ECSProvider{auth: auth}
}

var _ prvd.IInstance = &ECSProvider{}

type ECSProvider struct {
	auth *base.ClientMgr
}

func (e *ECSProvider) GetInstanceByIp(ip, region, vpc string) ([]ecs.Instance, error) {
	ips, err := json.Marshal([]string{ip})
	if err != nil {
		return nil, fmt.Errorf("get instances error: %s", err.Error())
	}
	req := ecs.CreateDescribeInstancesRequest()
	req.RegionId = region
	req.VpcId = vpc
	req.InstanceNetworkType = "vpc"
	req.PrivateIpAddresses = string(ips)
	req.NextToken = ""
	req.MaxResults = requests.NewInteger(50)

	var ecsInstances []ecs.Instance
	for {
		resp, err := e.auth.ECS.DescribeInstances(req)
		if err != nil {
			klog.Errorf("calling DescribeInstances: region=%s, "+
				"vpcId=%s, privateIpAddress=%s, message=[%s].", req.RegionId, req.VpcId, req.PrivateIpAddresses, err.Error())
			return nil, err
		}
		klog.V(5).Infof("RequestId: %s, API: %s, ips: %s", resp.RequestId, "DescribeInstances", string(ips))
		ecsInstances = append(ecsInstances, resp.Instances.Instance...)
		if resp.NextToken == "" {
			break
		}

		req.NextToken = resp.NextToken
	}

	return ecsInstances, nil
}

func (e *ECSProvider) DescribeNetworkInterfaces(vpcId string, ips []string, ipVersionType model.AddressIPVersionType) (map[string]string, error) {
	result := make(map[string]string)

	for begin := 0; begin < len(ips); begin += MaxNetworkInterfaceNum {
		last := len(ips)
		if begin+MaxNetworkInterfaceNum < last {
			last = begin + MaxNetworkInterfaceNum
		}
		privateIpAddress := ips[begin:last]

		req := ecs.CreateDescribeNetworkInterfacesRequest()
		req.VpcId = vpcId
		req.Status = "InUse"
		if ipVersionType == model.IPv6 {
			req.Ipv6Address = &privateIpAddress
		} else {
			req.PrivateIpAddress = &privateIpAddress
		}
		next := &util.Pagination{
			PageNumber: 1,
			PageSize:   100,
		}

		for {
			req.PageSize = requests.NewInteger(next.PageSize)
			req.PageNumber = requests.NewInteger(next.PageNumber)
			resp, err := e.auth.ECS.DescribeNetworkInterfaces(req)
			if err != nil {
				return result, err
			}
			klog.V(5).Infof("RequestId: %s, API: %s, ips: %s, privateIpAddress[%d:%d]",
				resp.RequestId, "DescribeNetworkInterfaces", privateIpAddress, begin, last)

			for _, eni := range resp.NetworkInterfaceSets.NetworkInterfaceSet {

				if ipVersionType == model.IPv6 {
					for _, ipv6 := range eni.Ipv6Sets.Ipv6Set {
						result[ipv6.Ipv6Address] = eni.NetworkInterfaceId
					}
				} else {
					for _, privateIp := range eni.PrivateIpSets.PrivateIpSet {
						result[privateIp.PrivateIpAddress] = eni.NetworkInterfaceId
					}
				}
			}

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

	}
	return result, nil
}
