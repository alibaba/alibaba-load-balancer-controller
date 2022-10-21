package backend

import (
	"context"
	"fmt"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"
	"k8s.io/klog/v2"

	"github.com/go-logr/logr"
	svchelper "k8s.io/alibaba-load-balancer-controller/pkg/controller/helper/service"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/store"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewBackendManager(store store.Storer, kubeClient client.Client, cloud prvd.Provider, logger logr.Logger) *Manager {
	return &Manager{
		store:            store,
		k8sClient:        kubeClient,
		EndpointResolver: NewDefaultEndpointResolver(store, kubeClient, cloud, logger),
	}
}

type Manager struct {
	store     store.Storer
	k8sClient client.Client
	EndpointResolver
}

func (mgr *Manager) BuildServicePortSDKBackends(ctx context.Context, svcKey types.NamespacedName, port intstr.IntOrString) ([]alb.BackendItem, bool, error) {
	svc := &v1.Service{}
	err := mgr.k8sClient.Get(context.Background(), svcKey, svc)
	if err != nil {
		return nil, false, err
	}

	var (
		modelBackends                   []alb.BackendItem
		endpoints                       []NodePortEndpoint
		containsPotentialReadyEndpoints bool
	)

	policy, err := svchelper.GetServiceTrafficPolicy(svc)
	if err != nil {
		return nil, false, err
	}
	switch policy {
	case svchelper.ENITrafficPolicy:
		endpoints, containsPotentialReadyEndpoints, err = mgr.ResolveENIEndpoints(ctx, util.NamespacedName(svc), port)
		if err != nil {
			return modelBackends, containsPotentialReadyEndpoints, err
		}
	case svchelper.LocalTrafficPolicy:
		endpoints, containsPotentialReadyEndpoints, err = mgr.ResolveLocalEndpoints(ctx, util.NamespacedName(svc), port)
		if err != nil {
			return modelBackends, containsPotentialReadyEndpoints, err
		}
	case svchelper.ClusterTrafficPolicy:
		endpoints, containsPotentialReadyEndpoints, err = mgr.ResolveClusterEndpoints(ctx, util.NamespacedName(svc), port)
		if err != nil {
			return modelBackends, containsPotentialReadyEndpoints, err
		}
	default:
		return modelBackends, containsPotentialReadyEndpoints, fmt.Errorf("not supported traffic policy [%s]", policy)
	}

	for _, endpoint := range endpoints {
		modelBackends = append(modelBackends, alb.BackendItem(endpoint))
	}

	return modelBackends, containsPotentialReadyEndpoints, nil
}

func (mgr *Manager) BuildServicePortsToSDKBackends(ctx context.Context, svc *v1.Service) (map[int32][]alb.BackendItem, bool, error) {
	svcPort2Backends := make(map[int32][]alb.BackendItem)
	containsPotentialReadyEndpoints := false
	for _, port := range svc.Spec.Ports {
		backends, _containsPotentialReadyEndpoints, err := mgr.BuildServicePortSDKBackends(ctx, util.NamespacedName(svc), intstr.FromInt(int(port.Port)))
		if err != nil {
			klog.Errorf("BuildServicePortsToSDKBackends: %v", err)
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, _containsPotentialReadyEndpoints, err
		}
		containsPotentialReadyEndpoints = _containsPotentialReadyEndpoints
		svcPort2Backends[port.Port] = backends
	}

	return svcPort2Backends, containsPotentialReadyEndpoints, nil
}
