package alibaba

import (
	"fmt"

	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/cas"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/ecs"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/nlb"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/slb"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/sls"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/vpc"
	"k8s.io/alibaba-load-balancer-controller/pkg/util/metric"
	"k8s.io/klog/v2"
)

func NewAlibabaCloud() prvd.Provider {
	mgr, err := base.NewClientMgr()
	if err != nil {
		panic(fmt.Sprintf("initialize alibaba cloud client auth: %s", err.Error()))
	}
	if mgr == nil {
		panic("auth should not be nil")
	}
	err = mgr.Start(base.RefreshToken)
	if err != nil {
		klog.Warningf("refresh token: %s", err.Error())
	}

	metric.RegisterPrometheus()

	return AlibabaCloud{
		IMetaData:   mgr.Meta,
		ECSProvider: ecs.NewECSProvider(mgr),
		SLBProvider: slb.NewLBProvider(mgr),
		VPCProvider: vpc.NewVPCProvider(mgr),
		ALBProvider: alb.NewALBProvider(mgr),
		NLBProvider: nlb.NewNLBProvider(mgr),
		SLSProvider: sls.NewSLSProvider(mgr),
		CASProvider: cas.NewCASProvider(mgr),
	}
}

var _ prvd.Provider = AlibabaCloud{}

type AlibabaCloud struct {
	*ecs.ECSProvider
	*vpc.VPCProvider
	*slb.SLBProvider
	*alb.ALBProvider
	*nlb.NLBProvider
	*sls.SLSProvider
	*cas.CASProvider
	prvd.IMetaData
}
