package framework

import (
	"context"
	"fmt"

	svc "k8s.io/alibaba-load-balancer-controller/pkg/controller/service"
	"k8s.io/alibaba-load-balancer-controller/pkg/model"
	nlbmodel "k8s.io/alibaba-load-balancer-controller/pkg/model/nlb"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/client"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/options"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

type ResourceType string

const (
	SLBResource = "SLB"
	NLBResource = "NLB"
	ACLResource = "ACL"
)

type Framework struct {
	Client          *client.E2EClient
	CreatedResource map[string]string
	PostCases       []func(f *Framework)
}

func NewFrameWork(c *client.E2EClient) *Framework {
	return &Framework{
		Client:          c,
		CreatedResource: make(map[string]string, 0),
		PostCases:       []func(f *Framework){},
	}
}

func (f *Framework) BeforeSuit() error {
	err := f.Client.KubeClient.CreateNamespace()
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = f.Client.KubeClient.DeleteService()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if err := f.Client.KubeClient.CreateDeployment(); err != nil {
		return err
	}

	if options.TestConfig.EnableVK {
		if err := f.Client.KubeClient.CreateVKDeployment(); err != nil {
			return err
		}
	}

	return nil
}

func (f *Framework) AfterSuit() error {
	for _, fc := range f.PostCases {
		fc(f)
	}
	err := f.Client.KubeClient.DeleteNamespace()
	if err != nil {
		return err
	}
	return f.CleanCloudResources()
}

func (f *Framework) AfterEach() error {
	_, err := f.Client.KubeClient.GetService()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err = f.Client.KubeClient.DeleteService(); err != nil {
		return err
	}
	return nil
}

func (f *Framework) CreateCloudResource() error {
	f.CreatedResource = make(map[string]string, 0)
	region, err := f.Client.CloudClient.Region()
	if err != nil {
		return err
	}

	zoneMappings, err := svc.ParseZoneMappings(options.TestConfig.NLBZoneMaps)
	if err != nil {
		return err
	}

	if options.TestConfig.InternetLoadBalancerID == "" {
		slbM := &model.LoadBalancer{
			LoadBalancerAttribute: model.LoadBalancerAttribute{
				AddressType:      model.InternetAddressType,
				LoadBalancerSpec: model.S1Small,
				RegionId:         region,
				LoadBalancerName: fmt.Sprintf("%s-%s-slb", options.TestConfig.ClusterId, "internet"),
			},
		}
		if err := f.Client.CloudClient.FindLoadBalancerByName(slbM); err != nil {
			return err
		}
		if slbM.LoadBalancerAttribute.LoadBalancerId == "" {
			if err := f.Client.CloudClient.CreateLoadBalancer(context.TODO(), slbM); err != nil {
				return fmt.Errorf("create internet slb error: %s", err.Error())
			}
		}
		options.TestConfig.InternetLoadBalancerID = slbM.LoadBalancerAttribute.LoadBalancerId
		f.CreatedResource[options.TestConfig.InternetLoadBalancerID] = SLBResource

		vsg1 := &model.VServerGroup{VGroupName: "test1"}
		err = f.Client.CloudClient.CreateVServerGroup(context.TODO(), vsg1, options.TestConfig.InternetLoadBalancerID)
		if err != nil {
			return fmt.Errorf("create vserver group error: %s", err.Error())
		}
		options.TestConfig.VServerGroupID = vsg1.VGroupId

		vsg2 := &model.VServerGroup{VGroupName: "test2"}
		err = f.Client.CloudClient.CreateVServerGroup(context.TODO(), vsg2, options.TestConfig.InternetLoadBalancerID)
		if err != nil {
			return fmt.Errorf("create vserver group error: %s", err.Error())
		}
		options.TestConfig.VServerGroupID2 = vsg2.VGroupId
	}

	if options.TestConfig.IntranetLoadBalancerID == "" {
		vswId, err := f.Client.CloudClient.VswitchID()
		if err != nil {
			return fmt.Errorf("get vsw id error: %s", err.Error())
		}
		slbM := &model.LoadBalancer{
			LoadBalancerAttribute: model.LoadBalancerAttribute{
				AddressType:      model.IntranetAddressType,
				LoadBalancerSpec: model.S1Small,
				RegionId:         region,
				VSwitchId:        vswId,
				LoadBalancerName: fmt.Sprintf("%s-%s-slb", options.TestConfig.ClusterId, "intranet"),
			},
		}
		if err := f.Client.CloudClient.FindLoadBalancerByName(slbM); err != nil {
			return err
		}
		if slbM.LoadBalancerAttribute.LoadBalancerId == "" {
			if err := f.Client.CloudClient.CreateLoadBalancer(context.TODO(), slbM); err != nil {
				return fmt.Errorf("create intranet slb error: %s", err.Error())
			}
		}
		options.TestConfig.IntranetLoadBalancerID = slbM.LoadBalancerAttribute.LoadBalancerId
		f.CreatedResource[options.TestConfig.IntranetLoadBalancerID] = SLBResource
	}

	if options.TestConfig.InternetNetworkLoadBalancerID == "" {
		slbM := &nlbmodel.NetworkLoadBalancer{
			LoadBalancerAttribute: &nlbmodel.LoadBalancerAttribute{
				AddressType:  nlbmodel.InternetAddressType,
				ZoneMappings: zoneMappings,
				VpcId:        options.TestConfig.VPCID,
				Name:         fmt.Sprintf("%s-%s-nlb", options.TestConfig.ClusterId, "internet"),
			},
		}

		if err := f.Client.CloudClient.FindNLBByName(context.TODO(), slbM); err != nil {
			return err
		}
		if slbM.LoadBalancerAttribute.LoadBalancerId == "" {
			if err := f.Client.CloudClient.CreateNLB(context.TODO(), slbM); err != nil {
				return fmt.Errorf("create internet nlb error: %s", err.Error())
			}
		}
		options.TestConfig.InternetNetworkLoadBalancerID = slbM.LoadBalancerAttribute.LoadBalancerId
		f.CreatedResource[options.TestConfig.InternetNetworkLoadBalancerID] = NLBResource
	}

	if options.TestConfig.IntranetNetworkLoadBalancerID == "" {
		slbM := &nlbmodel.NetworkLoadBalancer{
			LoadBalancerAttribute: &nlbmodel.LoadBalancerAttribute{
				AddressType:  nlbmodel.IntranetAddressType,
				ZoneMappings: zoneMappings,
				VpcId:        options.TestConfig.VPCID,
				Name:         fmt.Sprintf("%s-%s-nlb", options.TestConfig.ClusterId, "intranet"),
			},
		}

		if err := f.Client.CloudClient.FindNLBByName(context.TODO(), slbM); err != nil {
			return err
		}
		if slbM.LoadBalancerAttribute.LoadBalancerId == "" {
			if err := f.Client.CloudClient.CreateNLB(context.TODO(), slbM); err != nil {
				return fmt.Errorf("create intranet nlb error: %s", err.Error())
			}
		}
		options.TestConfig.IntranetNetworkLoadBalancerID = slbM.LoadBalancerAttribute.LoadBalancerId
		f.CreatedResource[options.TestConfig.IntranetNetworkLoadBalancerID] = NLBResource
	}

	if options.TestConfig.AclID == "" {
		aclName := fmt.Sprintf("%s-acl-%s", options.TestConfig.ClusterId, "a")
		aclId, err := f.Client.CloudClient.DescribeAccessControlList(context.TODO(), aclName)
		if err != nil {
			return fmt.Errorf("DescribeAccessControlList error: %s", err.Error())
		}
		if aclId == "" {
			aclId, err = f.Client.CloudClient.CreateAccessControlList(context.TODO(), aclName)
			if err != nil {
				return fmt.Errorf("CreateAccessControlList error: %s", err.Error())
			}
		}
		options.TestConfig.AclID = aclId
		f.CreatedResource[aclId] = ACLResource
	}

	if options.TestConfig.AclID2 == "" {
		aclName := fmt.Sprintf("%s-acl-%s", options.TestConfig.ClusterId, "b")
		aclId, err := f.Client.CloudClient.DescribeAccessControlList(context.TODO(), aclName)
		if err != nil {
			return fmt.Errorf("DescribeAccessControlList error: %s", err.Error())
		}
		if aclId == "" {
			aclId, err = f.Client.CloudClient.CreateAccessControlList(context.TODO(), aclName)
			if err != nil {
				return fmt.Errorf("CreateAccessControlList error: %s", err.Error())
			}
		}
		options.TestConfig.AclID2 = aclId
		f.CreatedResource[aclId] = ACLResource
	}

	klog.Infof("created resource: %s", util.PrettyJson(f.CreatedResource))
	return nil
}

func (f *Framework) DeleteLoadBalancer(lbid string) error {
	region, err := f.Client.CloudClient.Region()
	if err != nil {
		return err
	}
	slbM := &model.LoadBalancer{
		LoadBalancerAttribute: model.LoadBalancerAttribute{
			LoadBalancerId: lbid,
			RegionId:       region,
		},
	}
	err = f.Client.CloudClient.SetLoadBalancerDeleteProtection(context.TODO(), lbid, string(model.OffFlag))
	if err != nil {
		return err
	}

	err = f.Client.CloudClient.DeleteLoadBalancer(context.TODO(), slbM)
	if err != nil {
		return err
	}
	return nil
}

func (f *Framework) CleanCloudResources() error {
	klog.Infof("try to clean cloud resources: %+v", f.CreatedResource)
	for key, value := range f.CreatedResource {
		switch value {
		case SLBResource:
			if err := f.DeleteLoadBalancer(key); err != nil {
				return err
			}
		case NLBResource:
			if err := f.DeleteNetworkLoadBalancer(key); err != nil {
				return err
			}
		case ACLResource:
			if err := f.Client.CloudClient.DeleteAccessControlList(context.TODO(), key); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Framework) DeleteNetworkLoadBalancer(lbid string) error {
	slbM := &nlbmodel.NetworkLoadBalancer{
		LoadBalancerAttribute: &nlbmodel.LoadBalancerAttribute{
			LoadBalancerId: lbid,
		},
	}

	err := f.Client.CloudClient.DeleteNLB(context.TODO(), slbM)
	if err != nil {
		return err
	}
	return nil
}
