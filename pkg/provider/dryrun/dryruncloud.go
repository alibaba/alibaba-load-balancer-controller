package dryrun

import (
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/cas"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/ecs"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/nlb"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/sls"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/vpc"
	"k8s.io/klog/v2"
)

func NewDryRunCloud() prvd.Provider {
	auth, err := base.NewClientMgr()
	if err != nil {
		klog.Warningf("initialize alibaba cloud client auth: %s", err.Error())
	}
	if auth == nil {
		panic("auth should not be nil")
	}
	err = auth.Start(base.RefreshToken)
	if err != nil {
		klog.Warningf("refresh token: %s", err.Error())
	}

	cloud := &alibaba.AlibabaCloud{
		IMetaData:   auth.Meta,
		ECSProvider: ecs.NewECSProvider(auth),
		VPCProvider: vpc.NewVPCProvider(auth),
		ALBProvider: alb.NewALBProvider(auth),
		SLSProvider: sls.NewSLSProvider(auth),
		CASProvider: cas.NewCASProvider(auth),
		NLBProvider: nlb.NewNLBProvider(auth),
	}

	return &DryRunCloud{
		IMetaData: auth.Meta,
		DryRunECS: NewDryRunECS(auth, cloud.ECSProvider),
		DryRunVPC: NewDryRunVPC(auth, cloud.VPCProvider),
		DryRunALB: NewDryRunALB(auth, cloud.ALBProvider),
		DryRunSLS: NewDryRunSLS(auth, cloud.SLSProvider),
		DryRunCAS: NewDryRunCAS(auth, cloud.CASProvider),
		DryRunNLB: NewDryRunNLB(auth, cloud.NLBProvider),
	}
}

var _ prvd.Provider = &DryRunCloud{}

type DryRunCloud struct {
	*DryRunECS
	*DryRunVPC
	*DryRunALB
	*DryRunSLS
	*DryRunCAS
	*DryRunNLB
	prvd.IMetaData
}
