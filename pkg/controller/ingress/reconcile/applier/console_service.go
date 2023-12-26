package applier

import (
	"context"
	"fmt"
	"time"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"

	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	albmodel "k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
)

type ConsoleServiceManagerApplier interface {
	Apply(ctx context.Context, consoleServiceStack *albmodel.ConsoleServiceStack) error
}

var _ ConsoleServiceManagerApplier = &defaultConsoleServiceManagerApplier{}

func NewConsoleServiceManagerApplier(kubeClient client.Client, albProvider prvd.Provider, logger logr.Logger) *defaultConsoleServiceManagerApplier {
	return &defaultConsoleServiceManagerApplier{
		kubeClient:  kubeClient,
		albProvider: albProvider,
		logger:      logger,
	}
}

type defaultConsoleServiceManagerApplier struct {
	kubeClient  client.Client
	albProvider prvd.Provider

	logger logr.Logger
}

func (m *defaultConsoleServiceManagerApplier) Apply(ctx context.Context, consoleServiceStack *albmodel.ConsoleServiceStack) error {
	sdkSgp, err := m.albProvider.SelectALBServerGroupsByID(ctx, consoleServiceStack.ServerGroupID)
	if err != nil {
		m.logger.Error(err, "synthesize servers failed", "serverGroupID", consoleServiceStack.ServerGroupID)
		return err
	}

	if sdkSgp.Tags[util.ClusterNameTagKey] != "" && sdkSgp.Tags[util.ClusterNameTagKey] != consoleServiceStack.ClusterID {
		err := fmt.Errorf("ServerGroup managed by other cluster(current: %s, want: %s)", sdkSgp.Tags[util.ClusterNameTagKey], consoleServiceStack.ClusterID)
		m.logger.Error(err, "synthesize servers failed(ServerGroup managed by other cluster)", "serverGroupID", consoleServiceStack.ServerGroupID)
		return err
	}

	if sdkSgp.Tags[util.ClusterNameTagKey] == "" {
		m.tagConsoleService(ctx, consoleServiceStack)
	}

	serverApplier := NewServerApplier(m.kubeClient, m.albProvider, sdkSgp.ServerGroupId, consoleServiceStack.Backends, consoleServiceStack.TrafficPolicy, m.logger)
	if err := serverApplier.Apply(ctx); err != nil {
		m.logger.Error(err, "synthesize servers failed", "serverGroupID", consoleServiceStack.ServerGroupID)
		return err
	}
	return nil
}

func (m *defaultConsoleServiceManagerApplier) tagConsoleService(ctx context.Context, consoleServiceStack *albmodel.ConsoleServiceStack) error {
	traceID := ctx.Value(util.TraceID)
	tags := []albsdk.TagResourcesTag{
		{
			Key:   util.ClusterNameTagKey,
			Value: consoleServiceStack.ClusterID,
		},
		{
			Key:   util.ServiceNamespaceTagKey,
			Value: util.AvoidTagValueKeyword(consoleServiceStack.Namespace),
		},
		{
			Key:   util.ServiceNameTagKey,
			Value: util.AvoidTagValueKeyword(consoleServiceStack.Name),
		},
	}
	resIDs := []string{consoleServiceStack.ServerGroupID}
	tagReq := albsdk.CreateTagResourcesRequest()
	tagReq.Tag = &tags
	tagReq.ResourceId = &resIDs
	tagReq.ResourceType = util.ServerGroupResourceType
	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("tagging serverGroup",
		"traceID", traceID,
		"serverGroupID", consoleServiceStack.ServerGroupID,
		"startTime", startTime,
		util.Action, util.TagALBResource)
	tagResp, err := m.albProvider.TagALBResources(tagReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("tagged resource",
		"serverGroupID", consoleServiceStack.ServerGroupID,
		"requestID", tagResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.TagALBResource)
	return nil
}
