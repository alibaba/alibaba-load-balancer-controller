package alb

import (
	"context"
	"time"

	"github.com/onsi/gomega"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/framework"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

type ALB struct {
}

type Ingress struct {
}

func (*Ingress) waitCreateIngress(f *framework.Framework, ing *networkingv1.Ingress) {
	_, err := f.Client.KubeClient.CreateIngress(ing)
	klog.Info("apply ingress to apiserver: ingress=", ing, ", result=", err)
	gomega.Expect(err).To(gomega.BeNil())

	wait.Poll(5*time.Second, 2*time.Minute, func() (done bool, err error) {
		ingT, err := f.Client.KubeClient.NetworkingV1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, metav1.GetOptions{})
		if err != nil {
			klog.Infof("wait for ingress alb status ready: %s", err.Error())
			return false, nil
		}
		lb := ingT.Status.LoadBalancer

		if len(lb.Ingress) > 0 && lb.Ingress[0].Hostname != "" {
			return true, nil
		}
		klog.Infof("wait for ingress alb status not ready: %s", ingT.Name)
		return false, nil
	})
	ingN, err := f.Client.KubeClient.NetworkingV1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, metav1.GetOptions{})
	klog.Info("ingress reconcile by alb-ingress-controller, ingress=", ing, ", result=", err)
	gomega.Expect(err).To(gomega.BeNil())
	lb := ingN.Status.LoadBalancer
	dnsName := ""
	if len(lb.Ingress) > 0 && lb.Ingress[0].Hostname != "" {
		dnsName = lb.Ingress[0].Hostname
	}
	klog.Info("Expect ingress.Status.loadBalance.Ingress.Hostname != nil")
	gomega.Expect(dnsName).NotTo(gomega.BeEmpty(), printEventsWhenError(f))
}

// Rule func meaning
// DefaultRule--创建一条可以指定path或host的rule，path为空时，默认为/default-rule
type Rule struct {
}

func (*Rule) DefaultRule(path, host, serviceName string) networkingv1.IngressRule {
	if path == "" {
		path = "/default-rule"
	}
	exact := networkingv1.PathTypeExact
	ret := networkingv1.IngressRule{
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     path,
						PathType: &exact,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: serviceName,
								Port: networkingv1.ServiceBackendPort{
									Number: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	if host != "" {
		ret.Host = host
	}
	return ret
}
