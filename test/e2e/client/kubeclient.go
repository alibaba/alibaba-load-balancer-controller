package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/helper"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	Namespace        = "e2e-test"
	Service          = "basic-service"
	Secret           = "basic-secret"
	Deployment       = "nginx"
	VKDeployment     = "nginx-vk"
	NodeLabel        = "e2etest"
	ExcludeNodeLabel = "service.beta.kubernetes.io/exclude-node"
)

type KubeClient struct {
	kubernetes.Interface
}

func NewKubeClient(client kubernetes.Interface) *KubeClient {
	return &KubeClient{client}
}

// service
func (client *KubeClient) DefaultService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Service,
			Namespace: Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   v1.ProtocolTCP,
				},
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstr.FromInt(443),
					Protocol:   v1.ProtocolTCP,
				},
			},
			Type:            v1.ServiceTypeLoadBalancer,
			SessionAffinity: v1.ServiceAffinityNone,
			Selector:        map[string]string{"run": "nginx"},
		},
	}
}

// service
func (client *KubeClient) DefaultSecret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Secret,
			Namespace: Namespace,
		},
		Data: map[string][]byte{
			"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nMIIBazCCARKgAwIBAgIJAK1KWGXr2wjmMAkGByqGSM49BAEwJjEkMCIGA1UEAwwb\nYWxiLXRvcC5pbmdyZXNzLmFsaWJhYmEuY29tMB4XDTIyMDMyMzA3MTQxNFoXDTMy\nMDMyMDA3MTQxNFowJjEkMCIGA1UEAwwbYWxiLXRvcC5pbmdyZXNzLmFsaWJhYmEu\nY29tMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEZsHzfNnxRDhm0ZG+C8OlmlJU\nXyf3PTNrZrRx4wKIKHPuhZxxmAM/58Gf+ZlYVEZiymbok38MTIgdEfvYCYUpXaMq\nMCgwJgYDVR0RBB8wHYIbYWxiLXRvcC5pbmdyZXNzLmFsaWJhYmEuY29tMAkGByqG\nSM49BAEDSAAwRQIge+GbbNeEm0UhFobZPjr8sSDNMWwrqF/RszBPTQMzv7cCIQDH\n9I7i2WLBsW8wHIFy51oHNbbbMTL0PWD/QZ2LrFOhjQ==\n-----END CERTIFICATE-----\n"),
			"tls.key": []byte("-----BEGIN EC PARAMETERS-----\nBggqhkjOPQMBBw==\n-----END EC PARAMETERS-----\n-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIBzf5LCBwZAgJqFxacK9vA7OoFaKIxikk8xq5jgfpx3moAoGCCqGSM49\nAwEHoUQDQgAEZsHzfNnxRDhm0ZG+C8OlmlJUXyf3PTNrZrRx4wKIKHPuhZxxmAM/\n58Gf+ZlYVEZiymbok38MTIgdEfvYCYUpXQ==\n-----END EC PRIVATE KEY-----\n"),
		},
	}
}
func (client *KubeClient) IngressWithTLS(ing *networkingv1.Ingress, tlsNames []string) *networkingv1.Ingress {
	tls := []networkingv1.IngressTLS{
		{
			Hosts: tlsNames,
		},
	}
	ing.Spec.TLS = tls
	return ing
}

// service
func (client *KubeClient) DefaultIngress() *networkingv1.Ingress {
	ingressClassName := "alb"
	pathTypePrefix := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Service + "-ingress",
			Namespace: Namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: "alb.ingress.alibaba.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{

											Name: Service,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (client *KubeClient) DefaultIngressWithSvcName(svcName string) *networkingv1.Ingress {
	ingressClassName := "alb"
	pathTypePrefix := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName + "-ingress",
			Namespace: Namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: "alb.ingress.alibaba.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{

											Name: svcName,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (client *KubeClient) CreateServiceByAnno(anno map[string]string) (*v1.Service, error) {
	svc := client.DefaultService()
	svc.Annotations = anno
	return client.CoreV1().Services(Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
}

func (client *KubeClient) CreateNLBServiceByAnno(anno map[string]string) (*v1.Service, error) {
	svc := client.DefaultService()
	lbClass := helper.NLBClass
	svc.Spec.LoadBalancerClass = &lbClass
	svc.Annotations = anno

	return client.CoreV1().Services(Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
}

func (client *KubeClient) CreateServiceWithStringTargetPort(anno map[string]string) (*v1.Service, error) {
	svc := client.DefaultService()
	svc.Annotations = anno
	svc.Spec.Ports = []v1.ServicePort{
		{
			Name:       "http",
			Port:       80,
			TargetPort: intstr.FromString("http"),
			Protocol:   v1.ProtocolTCP,
		},
		{
			Name:       "https",
			Port:       443,
			TargetPort: intstr.FromString("https"),
			Protocol:   v1.ProtocolTCP,
		},
	}
	return client.CoreV1().Services(Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
}

func (client *KubeClient) CreateIngress(ing *networkingv1.Ingress) (*networkingv1.Ingress, error) {
	if ing == nil {
		return nil, fmt.Errorf("ingress is nil")
	}
	return client.NetworkingV1().Ingresses(Namespace).Create(context.TODO(), ing, metav1.CreateOptions{})
}

func (client *KubeClient) CreateNLBServiceWithStringTargetPort(anno map[string]string) (*v1.Service, error) {
	lbClass := helper.NLBClass
	svc := client.DefaultService()
	svc.Annotations = anno
	svc.Spec.LoadBalancerClass = &lbClass
	svc.Spec.Ports = []v1.ServicePort{
		{
			Name:       "http",
			Port:       80,
			TargetPort: intstr.FromString("http"),
			Protocol:   v1.ProtocolTCP,
		},
		{
			Name:       "https",
			Port:       443,
			TargetPort: intstr.FromString("https"),
			Protocol:   v1.ProtocolTCP,
		},
	}
	return client.CoreV1().Services(Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
}

func (client *KubeClient) CreateService(svc *v1.Service) (*v1.Service, error) {
	if svc == nil {
		return nil, fmt.Errorf("svc is nil")
	}
	return client.CoreV1().Services(Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
}

func (client *KubeClient) PatchService(oldSvc, newSvc *v1.Service) (*v1.Service, error) {
	oldStr, _ := json.Marshal(oldSvc)
	newStr, _ := json.Marshal(newSvc)
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldStr, newStr, &v1.Service{})
	if patchErr != nil {
		return nil, fmt.Errorf("create merge patch: %s", patchErr.Error())
	}
	return client.CoreV1().Services(Namespace).Patch(context.TODO(), Service, types.StrategicMergePatchType,
		patchBytes, metav1.PatchOptions{})
}

func (client *KubeClient) CreateServiceWithoutSelector(anno map[string]string) (*v1.Service, error) {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        Service,
			Namespace:   Namespace,
			Annotations: anno,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   v1.ProtocolTCP,
				},
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstr.FromInt(80),
					Protocol:   v1.ProtocolTCP,
				},
			},
			Type: v1.ServiceTypeLoadBalancer,
		},
	}

	return client.CoreV1().Services(Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
}

func (client *KubeClient) CreateNLBServiceWithoutSelector(anno map[string]string) (*v1.Service, error) {
	lbClass := helper.NLBClass
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        Service,
			Namespace:   Namespace,
			Annotations: anno,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   v1.ProtocolTCP,
				},
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstr.FromInt(80),
					Protocol:   v1.ProtocolTCP,
				},
			},
			Type:              v1.ServiceTypeLoadBalancer,
			LoadBalancerClass: &lbClass,
		},
	}

	return client.CoreV1().Services(Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
}

func (client *KubeClient) DeleteService() error {
	return wait.PollImmediate(3*time.Second, 5*time.Minute, func() (done bool, err error) {
		err = client.CoreV1().Services(Namespace).Delete(context.TODO(), Service, metav1.DeleteOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
		}
		return false, nil
	})
}

func (client *KubeClient) DeleteServiceByName(name string) error {
	return wait.PollImmediate(3*time.Second, 3*time.Minute, func() (done bool, err error) {
		err = client.CoreV1().Services(Namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
		}
		return false, nil
	})
}

func (client *KubeClient) GetService() (*v1.Service, error) {
	return client.CoreV1().Services(Namespace).Get(context.TODO(), Service, metav1.GetOptions{})
}

// endpoints

func (client *KubeClient) GetEndpoint() (*v1.Endpoints, error) {
	return client.CoreV1().Endpoints(Namespace).Get(context.TODO(), Service, metav1.GetOptions{})
}

func (client *KubeClient) CreateEndpointsWithoutNodeName() (*v1.Endpoints, error) {
	ep := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Service,
			Namespace: Namespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "123.123.123.123",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Port:     80,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}
	return client.CoreV1().Endpoints(Namespace).Create(context.TODO(), ep, metav1.CreateOptions{})
}

func (client *KubeClient) CreateEndpointsWithNotExistNode() (*v1.Endpoints, error) {
	nodeName := "cn-hangzhou.123.123.123.123"
	ep := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Service,
			Namespace: Namespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "123.123.123.123",
						NodeName: &nodeName,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Port:     80,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}
	return client.CoreV1().Endpoints(Namespace).Create(context.TODO(), ep, metav1.CreateOptions{})
}

// deployment
func (client *KubeClient) CreateDeployment() error {
	var replica int32 = 3
	test_image := "ack-asi-ci-registry.cn-hongkong.cr.aliyuncs.com/chorus-public/nginx-multiarch:latest"
	if IsDomesticRegion() {
		test_image = "ack-asi-ci-registry.cn-hangzhou.cr.aliyuncs.com/chorus-public/nginx-multiarch:latest"
	}

	nginx := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Deployment,
			Namespace: Namespace,
			Labels: map[string]string{
				"run": "nginx",
				"app": "nginx",
			},
		},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"run": "nginx",
					"app": "nginx",
				},
			},
			Replicas: &replica,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"run": "nginx",
						"app": "nginx",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "nginx",
							Image:           test_image,
							ImagePullPolicy: v1.PullIfNotPresent,
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      v1.ProtocolTCP,
								},
								{
									Name:          "https",
									ContainerPort: 443,
									Protocol:      v1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := client.AppsV1().Deployments(Namespace).Create(context.Background(), nginx, metav1.CreateOptions{})
	if err != nil {
		if !strings.Contains(err.Error(), "exists") {
			return fmt.Errorf("create nginx error: %s", err.Error())
		}
	}
	return wait.Poll(5*time.Second, 2*time.Minute, func() (done bool, err error) {
		pods, err := client.CoreV1().Pods(nginx.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "run=nginx"})
		if err != nil {
			klog.Infof("wait for nginx pod ready: %s", err.Error())
			return false, nil
		}
		if len(pods.Items) != int(*nginx.Spec.Replicas) {
			klog.Infof("wait for nginx pod replicas: %d", len(pods.Items))
			return false, nil
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" {
				klog.Infof("wait for nginx pod Running: %s", pod.Name)
				return false, nil
			}
		}
		return true, nil
	},
	)
}

func (client *KubeClient) ScaleDeployment(replica int32) error {
	deploy, err := client.AppsV1().Deployments(Namespace).Get(context.TODO(), Deployment, metav1.GetOptions{})
	if err != nil {
		return err
	}
	deploy.Spec.Replicas = &replica
	_, err = client.AppsV1().Deployments(Namespace).Update(context.TODO(), deploy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return wait.PollImmediate(5*time.Second, 2*time.Minute, func() (done bool, err error) {
		pods, err := client.CoreV1().Pods(Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "run=nginx"})
		if err != nil {
			klog.Infof("wait for nginx pod ready: %s", err.Error())
			return false, nil
		}
		if len(pods.Items) != int(replica) {
			klog.Infof("wait for nginx pod replicas: %d", len(pods.Items))
			return false, nil
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" {
				klog.Infof("wait for nginx pod Running: %s", pod.Name)
				return false, nil
			}
		}
		return true, nil
	})
}

func (client *KubeClient) CreateVKDeployment() error {
	var replica int32 = 2
	nginx := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      VKDeployment,
			Namespace: Namespace,
			Labels: map[string]string{
				"run": "nginx",
				"app": "nginx-vk",
			},
		},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"run": "nginx",
					"app": "nginx-vk",
				},
			},
			Replicas: &replica,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"run": "nginx",
						"app": "nginx-vk",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "nginx",
							Image:           "nginx:1.9.7",
							ImagePullPolicy: "Always",
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      v1.ProtocolTCP,
								},
								{
									Name:          "https",
									ContainerPort: 443,
									Protocol:      v1.ProtocolTCP,
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"type": "virtual-kubelet",
					},
					Tolerations: []v1.Toleration{
						{
							Operator: v1.TolerationOpExists,
						},
					},
				},
			},
		},
	}

	_, err := client.AppsV1().Deployments(Namespace).Create(context.Background(), nginx, metav1.CreateOptions{})
	if err != nil {
		if !strings.Contains(err.Error(), "exists") {
			return fmt.Errorf("create nginx error: %s", err.Error())
		}
	}
	return wait.Poll(5*time.Second, 2*time.Minute, func() (done bool, err error) {
		pods, err := client.CoreV1().Pods(nginx.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "app=nginx-vk"})
		if err != nil {
			klog.Infof("wait for nginx pod ready: %s", err.Error())
			return false, nil
		}
		if len(pods.Items) != int(*nginx.Spec.Replicas) {
			klog.Infof("wait for nginx pod replicas: %d", len(pods.Items))
			return false, nil
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" {
				klog.Infof("wait for nginx pod Running: %s", pod.Name)
				return false, nil
			}
		}
		return true, nil
	},
	)

}

// namespace
func (client *KubeClient) CreateNamespace() error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Namespace,
			Namespace: Namespace,
		},
	}
	_, err := client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	return err
}

func (client *KubeClient) DeleteNamespace() error {
	return wait.PollImmediate(5*time.Second, 3*time.Minute,
		func() (done bool, err error) {
			err = client.CoreV1().Namespaces().Delete(context.TODO(), Namespace, metav1.DeleteOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return true, nil
				}
			}
			return false, nil
		})
}

// node
func (client *KubeClient) LabelNode(nodeName string, key string, value string) error {
	return wait.PollImmediate(2*time.Second, time.Minute, func() (done bool, err error) {
		n, err := client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		n.ObjectMeta.Labels[key] = value
		_, err = client.CoreV1().Nodes().Update(context.TODO(), n, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (client *KubeClient) UnLabelNode(nodeName string, key string) error {
	return wait.PollImmediate(2*time.Second, time.Minute, func() (done bool, err error) {
		n, err := client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		delete(n.ObjectMeta.Labels, key)
		_, err = client.CoreV1().Nodes().Update(context.TODO(), n, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (client *KubeClient) UnscheduledNode(nodeName string) error {
	return wait.PollImmediate(2*time.Second, time.Minute, func() (done bool, err error) {
		n, err := client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err != nil || n == nil {
			return false, nil
		}
		n.Spec.Unschedulable = true
		_, err = client.CoreV1().Nodes().Update(context.TODO(), n, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})

}

func (client *KubeClient) ScheduledNode(nodeName string) error {
	return wait.PollImmediate(2*time.Second, time.Minute, func() (done bool, err error) {
		n, err := client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err != nil || n == nil {
			return false, nil
		}
		n.Spec.Unschedulable = false
		_, err = client.CoreV1().Nodes().Update(context.TODO(), n, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})

}

func (client *KubeClient) AddTaint(nodeName string, taint v1.Taint) error {
	return wait.PollImmediate(2*time.Second, 30*time.Second, func() (done bool, err error) {
		n, err := client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		for _, taint := range n.Spec.Taints {
			if taint.Key == taint.Key {
				return true, nil
			}
		}
		n.Spec.Taints = append(n.Spec.Taints, taint)
		_, err = client.CoreV1().Nodes().Update(context.TODO(), n, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (client *KubeClient) RemoveTaint(nodeName string, taint v1.Taint) error {
	return wait.PollImmediate(2*time.Second, 30*time.Second, func() (done bool, err error) {
		n, err := client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		var updateTaints []v1.Taint
		for _, t := range n.Spec.Taints {
			if t.Key == taint.Key {
				continue
			}
			updateTaints = append(updateTaints, t)
		}
		_, err = client.CoreV1().Nodes().Update(context.TODO(), n, metav1.UpdateOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (client *KubeClient) ListNodes() ([]v1.Node, error) {
	nodeList, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func (client *KubeClient) GetLatestNode() (*v1.Node, error) {
	nodeList, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(nodeList.Items) == 0 {
		return nil, nil
	}

	var ret v1.Node
	for _, node := range nodeList.Items {
		if helper.HasExcludeLabel(&node) {
			continue
		}
		if _, exclude := node.Labels[helper.LabelNodeExcludeBalancer]; exclude {
			continue
		}
		if _, isVK := node.Labels[helper.LabelNodeTypeVK]; isVK {
			continue
		}
		if ret.Name == "" {
			ret = node
		} else if ret.CreationTimestamp.Before(&node.CreationTimestamp) {
			ret = node
		}
	}
	klog.Infof("return node:%s", ret.Name)
	return &ret, nil
}

func (client *KubeClient) PatchNodeStatus(oldNode, newNode *v1.Node) (*v1.Node, error) {
	oldStr, _ := json.Marshal(oldNode)
	newStr, _ := json.Marshal(newNode)
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldStr, newStr, &v1.Node{})
	if patchErr != nil {
		return nil, fmt.Errorf("create merge patch: %s", patchErr.Error())
	}
	return client.CoreV1().Nodes().PatchStatus(context.TODO(), oldNode.Name, patchBytes)
}

func (client *KubeClient) PatchNode(oldNode, newNode *v1.Node) (*v1.Node, error) {
	oldStr, _ := json.Marshal(oldNode)
	newStr, _ := json.Marshal(newNode)
	patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldStr, newStr, &v1.Node{})
	if patchErr != nil {
		return nil, fmt.Errorf("create merge patch: %s", patchErr.Error())
	}
	return client.CoreV1().Nodes().Patch(context.TODO(), oldNode.Name,
		types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
}
