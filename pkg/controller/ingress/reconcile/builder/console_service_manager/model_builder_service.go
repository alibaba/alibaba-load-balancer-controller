package servicemanager

import (
	"context"
	"fmt"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/helper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/annotations"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/backend"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Builder interface {
	Build(ctx context.Context, svc *corev1.Service, clusterId string) (*alb.ConsoleServiceStack, error)
}

var _ Builder = &defaultConsoleServiceManagerBuilder{}

type defaultConsoleServiceManagerBuilder struct {
	backendMgr *backend.Manager
	k8sClient  client.Client
}

func NewDefaultServiceStackBuilder(backendMgr *backend.Manager, k8sClient client.Client) *defaultConsoleServiceManagerBuilder {
	return &defaultConsoleServiceManagerBuilder{
		backendMgr: backendMgr,
		k8sClient:  k8sClient,
	}
}

func (s defaultConsoleServiceManagerBuilder) Build(ctx context.Context, svc *corev1.Service, clusterId string) (*alb.ConsoleServiceStack, error) {
	serviceStack := &alb.ConsoleServiceStack{}
	serviceStack.ClusterID = clusterId
	serviceStack.ServerGroupID = svc.Annotations[annotations.AlbServerGroupId]
	serviceStack.Namespace = svc.Namespace
	serviceStack.Name = svc.Name
	serviceStack.ContainsPotentialReadyEndpoints = false
	serviceStack.Backends = []alb.BackendItem{}
	service := &corev1.Service{}
	if err := s.k8sClient.Get(ctx, types.NamespacedName{
		Namespace: svc.Namespace,
		Name:      svc.Name,
	}, service); err != nil && !errors.IsNotFound(err) {
		return nil, err
	} else {
		policy, err := helper.GetServiceTrafficPolicy(svc)
		if err != nil {
			return nil, err
		}
		serviceStack.TrafficPolicy = string(policy)
		port2backends, containsPotentialReadyEndpoints, err := s.backendMgr.BuildServicePortsToSDKBackends(ctx, svc)
		if err != nil {
			return nil, fmt.Errorf("build serviceToServerGroup error: %v", err)
		}
		if len(port2backends) > 1 {
			return nil, fmt.Errorf("not supportservices using multiple ports")
		}
		for _, backends := range port2backends {
			serviceStack.Backends = append(serviceStack.Backends, backends...)
		}
		serviceStack.ContainsPotentialReadyEndpoints = containsPotentialReadyEndpoints
	}
	return serviceStack, nil
}
