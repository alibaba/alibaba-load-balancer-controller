package applier

import (
	"context"
	"fmt"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/tracking"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/store"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/backend"
	albmodel "k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb/core"
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
)

type AlbConfigManagerApplier interface {
	Apply(ctx context.Context, stack core.Manager) error
}

var _ AlbConfigManagerApplier = &defaultAlbConfigManagerApplier{}

func NewAlbConfigManagerApplier(store store.Storer, kubeClient client.Client, provider prvd.Provider, tagPrefix string, logger logr.Logger) *defaultAlbConfigManagerApplier {
	trackingProvider := tracking.NewDefaultProvider(tagPrefix, provider.ClusterID())
	backendManager := backend.NewBackendManager(store, kubeClient, provider, logger)
	return &defaultAlbConfigManagerApplier{
		trackingProvider: trackingProvider,
		backendManager:   *backendManager,
		kubeClient:       kubeClient,
		albProvider:      provider,
		logger:           logger,
	}
}

type defaultAlbConfigManagerApplier struct {
	kubeClient client.Client

	trackingProvider tracking.TrackingProvider
	backendManager   backend.Manager
	albProvider      prvd.Provider

	logger logr.Logger
}

type ResourceApply interface {
	Apply(ctx context.Context) error
	PostApply(ctx context.Context) error
}

func (m *defaultAlbConfigManagerApplier) Apply(ctx context.Context, stack core.Manager) error {

	// Reuse LoadBalancer
	var resLBs []*albmodel.AlbLoadBalancer
	_ = stack.ListResources(&resLBs)
	if len(resLBs) > 1 {
		return fmt.Errorf("invalid res loadBalancers, at most one loadBalancer for stack: %s", stack.StackID())
	}
	// Reuse LoadBalancer
	var isReuseLb bool
	if v, ok := ctx.Value(util.IsReuseLb).(bool); ok {
		isReuseLb = v
	}
	commonReuse := false
	// reuse=true, forceOverride=false => commonReuse=true
	if isReuseLb && len(resLBs) == 1 && resLBs[0].Spec.ForceOverride != nil && !*resLBs[0].Spec.ForceOverride {
		commonReuse = true
	}
	// only loadbalaner apply if delete albconfig
	if len(resLBs) == 0 {
		applier := NewAlbLoadBalancerApplier(m.albProvider, m.trackingProvider, stack, m.logger, commonReuse)
		return applier.Apply(ctx)
	}

	appliers := []ResourceApply{
		NewSecretApplier(m.albProvider, stack, m.logger),
		NewServerGroupApplier(m.kubeClient, m.backendManager, m.albProvider, m.trackingProvider, stack, m.logger),
		NewAlbLoadBalancerApplier(m.albProvider, m.trackingProvider, stack, m.logger, commonReuse),
		NewListenerApplier(m.albProvider, stack, m.logger, commonReuse),
		NewAclApplier(m.albProvider, m.trackingProvider, stack, m.logger),
		NewListenerRuleApplier(m.albProvider, stack, m.logger),
	}

	for _, applier := range appliers {
		if err := applier.Apply(ctx); err != nil {
			return err
		}
	}

	for i := len(appliers) - 1; i >= 0; i-- {
		if err := appliers[i].PostApply(ctx); err != nil {
			return err
		}
	}

	return nil
}
