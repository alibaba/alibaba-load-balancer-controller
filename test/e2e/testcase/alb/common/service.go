package common

import (
	"context"

	"github.com/onsi/gomega"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/framework"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// Service
// createDefaultService
type Service struct {
}

func (*Service) CreateDefaultService(f *framework.Framework) {
	svc := f.Client.KubeClient.DefaultService()
	svc.Spec.Type = v1.ServiceTypeNodePort
	_, err := f.Client.KubeClient.CoreV1().Services(svc.Namespace).Get(context.TODO(), svc.Name, metav1.GetOptions{})
	if err != nil {
		_, err = f.Client.KubeClient.CreateService(svc)
	}
	gomega.Expect(err).To(gomega.BeNil())
	klog.Infof("node port service created :%s/%s", svc.Namespace, svc.Name)
}
