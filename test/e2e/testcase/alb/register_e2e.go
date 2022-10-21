package alb

import (
	"k8s.io/alibaba-load-balancer-controller/test/e2e/framework"
	"k8s.io/klog/v2"
)

type runFunc func(f *framework.Framework)

type RegisterCaseE2E struct {
	f      runFunc
	reason string
	author string
	flag   string
}

var e2eCases []RegisterCaseE2E

func InSlice(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

func InitAlbIngressE2ECases() {
	e2eCases = append(e2eCases, RegisterCaseE2E{
		f:      RunIngressTestCases,
		reason: "basic ingress test",
		author: "yuri",
		flag:   "basic",
	})

	// e2eCases = append(e2eCases, RegisterCaseE2E{
	// 	f:      RunCustomizeConditionTestCases,
	// 	reason: "test customize condition",
	// 	author: "yuri",
	// 	flag:   "customize-condition",
	// })
}

func ExecuteIngressE2ECases(frame *framework.Framework, albFlags []string) {
	klog.Info(e2eCases)
	for n, e2eFunc := range e2eCases {
		klog.Info(albFlags)
		klog.Info(len(albFlags) != 0)
		klog.Info(!InSlice(albFlags, e2eFunc.flag))
		klog.Infof("ExecuteIngressE2ECases %d %s created by %s", n, e2eFunc.reason, e2eFunc.author)
		e2eFunc.f(frame)
	}
}
