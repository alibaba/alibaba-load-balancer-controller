package alb

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/tracking"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"

	"github.com/go-logr/logr"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/alb/future"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	DefaultWaitSGPDeletionPollInterval  = 2 * time.Second
	DefaultWaitSGPDeletionTimeout       = 50 * time.Second
	DefaultWaitLSExistencePollInterval  = 2 * time.Second
	DefaultWaitLSExistenceTimeout       = 20 * time.Second
	DefaultWaitAclExistencePollInterval = 2 * time.Second
	DefaultWaitAclExistenceTimeout      = 20 * time.Second
)

type ResourceType = string

const (
	loadBalancerResourceType   ResourceType = "loadbalancer"
	serverGroupResourceType    ResourceType = "servergroup"
	acResourceType             ResourceType = "acl"
	securityPolicyResourceType ResourceType = "securitypolicy"
)

func NewALBProvider(
	auth *base.ClientMgr,
) *ALBProvider {
	logger := ctrl.Log.WithName("controllers").WithName("ALBProvider")
	listenerIdSdkCertsMap := make(map[string][]albsdk.CertificateModel)
	return &ALBProvider{
		logger:                       logger,
		auth:                         auth,
		promise:                      future.NewPromise(),
		listenerIdSdkCertsMap:        listenerIdSdkCertsMap,
		waitSGPDeletionPollInterval:  DefaultWaitSGPDeletionPollInterval,
		waitSGPDeletionTimeout:       DefaultWaitSGPDeletionTimeout,
		waitLSExistenceTimeout:       DefaultWaitLSExistenceTimeout,
		waitLSExistencePollInterval:  DefaultWaitLSExistencePollInterval,
		waitAclExistencePollInterval: DefaultWaitAclExistencePollInterval,
		waitAclExistenceTimeout:      DefaultWaitAclExistenceTimeout,
	}
}

var _ prvd.IALB = &ALBProvider{}

type ALBProvider struct {
	auth                         *base.ClientMgr
	logger                       logr.Logger
	promise                      future.Promise
	listenerIdSdkCertsMap        map[string][]albsdk.CertificateModel
	waitLSExistencePollInterval  time.Duration
	waitLSExistenceTimeout       time.Duration
	waitSGPDeletionPollInterval  time.Duration
	waitSGPDeletionTimeout       time.Duration
	waitAclExistencePollInterval time.Duration
	waitAclExistenceTimeout      time.Duration
}

func (m *ALBProvider) DoAction(request requests.AcsRequest, response responses.AcsResponse) (err error) {
	return m.auth.ALB.Client.DoAction(request, response)
}

func (m *ALBProvider) CreateALB(ctx context.Context, resLB *alb.AlbLoadBalancer, trackingProvider tracking.TrackingProvider) (alb.LoadBalancerStatus, error) {
	traceID := ctx.Value(util.TraceID)

	createLbReq, err := buildSDKCreateAlbLoadBalancerRequest(resLB.Spec)
	if err != nil {
		return alb.LoadBalancerStatus{}, err
	}

	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("creating loadBalancer",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"startTime", startTime,
		util.Action, util.CreateALBLoadBalancer)
	createLbResp, err := m.auth.ALB.CreateLoadBalancer(createLbReq)
	if err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	m.logger.V(util.MgrLogLevel).Info("created loadBalancer",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"loadBalancerID", createLbResp.LoadBalancerId,
		"traceID", traceID,
		"requestID", createLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.CreateALBLoadBalancer)

	asynchronousStartTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("creating loadBalancer asynchronous",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", createLbResp.LoadBalancerId,
		"startTime", asynchronousStartTime,
		util.Action, util.CreateALBLoadBalancerAsynchronous)
	var getLbResp *albsdk.GetLoadBalancerAttributeResponse
	for i := 0; i < util.CreateLoadBalancerWaitActiveMaxRetryTimes; i++ {
		time.Sleep(util.CreateLoadBalancerWaitActiveRetryInterval)

		getLbResp, err = getALBLoadBalancerAttributeFunc(ctx, createLbResp.LoadBalancerId, m.auth, m.logger)
		if err != nil {
			return alb.LoadBalancerStatus{}, err
		}
		if isAlbLoadBalancerActive(getLbResp.LoadBalancerStatus) {
			break
		}
	}
	m.logger.V(util.MgrLogLevel).Info("created loadBalancer asynchronous",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", createLbResp.LoadBalancerId,
		"requestID", getLbResp.RequestId,
		"elapsedTime", time.Since(asynchronousStartTime).Milliseconds(),
		util.Action, util.CreateALBLoadBalancerAsynchronous)

	if err := m.Tag(ctx, resLB, createLbResp.LoadBalancerId, trackingProvider); err != nil {
		if errTmp := m.DeleteALB(ctx, createLbResp.LoadBalancerId); errTmp != nil {
			m.logger.V(util.MgrLogLevel).Error(errTmp, "roll back load balancer failed",
				"stackID", resLB.Stack().StackID(),
				"resourceID", resLB.ID(),
				"traceID", traceID,
				"loadBalancerID", createLbResp.LoadBalancerId,
				util.Action, util.TagALBResource)
		}
		return alb.LoadBalancerStatus{}, err
	}
	// if err := m.ServiceManagedControl(ctx, resLB, createLbResp.LoadBalancerId); err != nil {
	// 	m.logger.V(util.MgrLogLevel).Error(err, "service managed control resource failed",
	// 		"stackID", resLB.Stack().StackID(),
	// 		"resourceID", resLB.ID(),
	// 		"traceID", traceID,
	// 		"loadBalancerID", createLbResp.LoadBalancerId,
	// 		"elapsedTime", time.Since(asynchronousStartTime).Milliseconds(),
	// 		util.Action, util.ALBInnerServiceManagedControl)
	// 	if errTmp := m.DeleteALB(ctx, createLbResp.LoadBalancerId); errTmp != nil {
	// 		m.logger.V(util.MgrLogLevel).Error(errTmp, "roll back load balancer failed",
	// 			"stackID", resLB.Stack().StackID(),
	// 			"resourceID", resLB.ID(),
	// 			"traceID", traceID,
	// 			"loadBalancerID", createLbResp.LoadBalancerId,
	// 			util.Action, util.ALBInnerServiceManagedControl)
	// 	}
	// 	return alb.LoadBalancerStatus{}, err
	// }

	if resLB.Spec.Ipv6AddressType == util.LoadBalancerIpv6AddressTypeInternet {
		if _, err := enableLoadBalancerIpv6InternetFunc(ctx, createLbResp.LoadBalancerId, m.auth, m.logger); err != nil {
			return alb.LoadBalancerStatus{}, err
		}
	}

	if len(resLB.Spec.AccessLogConfig.LogProject) != 0 && len(resLB.Spec.AccessLogConfig.LogStore) != 0 {
		if err := m.AnalyzeAndAssociateAccessLogToALB(ctx, createLbResp.LoadBalancerId, resLB); err != nil {
			return alb.LoadBalancerStatus{}, err
		}
	}

	return buildResAlbLoadBalancerStatus(createLbResp.LoadBalancerId, getLbResp.DNSName), nil
}

func isAlbLoadBalancerActive(status string) bool {
	return strings.EqualFold(status, util.LoadBalancerStatusActive)
}

func (m *ALBProvider) MoveResourceGroup(ctx context.Context, resType ResourceType, resourceId, newResourceGroupId string) error {
	traceID := ctx.Value(util.TraceID)

	moveResourceGroupRequest := albsdk.CreateMoveResourceGroupRequest()
	moveResourceGroupRequest.ResourceId = resourceId
	moveResourceGroupRequest.ResourceType = resType
	moveResourceGroupRequest.NewResourceGroupId = newResourceGroupId

	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("moveResourceGroup",
		"resourceID", resourceId,
		"traceID", traceID,
		"ResourceType", resType,
		"NewResourceGroupId", newResourceGroupId,
		"startTime", startTime,
		util.Action, util.MoveResourceGroup)

	moveResGreoupResp, err := m.auth.ALB.MoveResourceGroup(moveResourceGroupRequest)
	if err != nil {
		return err
	}

	m.logger.V(util.MgrLogLevel).Info("moveResourceGroup",
		"resourceID", resourceId,
		"traceID", traceID,
		"ResourceType", resType,
		"NewResourceGroupId", newResourceGroupId,
		"requestID", moveResGreoupResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.MoveResourceGroup)
	return nil
}

func (m *ALBProvider) updateAlbLoadBalancerResourceGroup(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	if resLB.Spec.ResourceGroupId == "" {
		return nil
	}
	traceID := ctx.Value(util.TraceID)
	if resLB.Spec.ResourceGroupId == sdkLB.ResourceGroupId {
		return nil
	}
	m.logger.V(util.MgrLogLevel).Info("AlbLoadBalancerResourceGroup need update",
		"sdk", sdkLB.Tags,
		"oldResourceGroupId", sdkLB.ResourceGroupId,
		"newResourceGroupId", resLB.Spec.ResourceGroupId,
		"loadBalancerID", sdkLB.LoadBalancerId,
		"traceID", traceID)
	if err := m.MoveResourceGroup(ctx, loadBalancerResourceType, sdkLB.LoadBalancerId, resLB.Spec.ResourceGroupId); err != nil {
		return err
	}
	return nil
}

func (m *ALBProvider) updateAlbLoadBalancerAddressTypeConfig(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)
	if len(resLB.Spec.LoadBalancerId) != 0 {
		return nil
	}
	var (
		isLoadBalancerAddressTypeNeedUpdate = false
	)

	if !isAlbLoadBalancerAddressTypeValid(resLB.Spec.AddressType) {
		return fmt.Errorf("invalid load balancer address type: %s", resLB.Spec.AddressType)
	}

	if !strings.EqualFold(resLB.Spec.AddressType, sdkLB.AddressType) {
		m.logger.V(util.MgrLogLevel).Info("LoadBalancer AddressType update",
			"res", resLB.Spec.AddressType,
			"sdk", sdkLB.AddressType,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		isLoadBalancerAddressTypeNeedUpdate = true
	}
	if !isLoadBalancerAddressTypeNeedUpdate {
		return nil
	}

	updateLbReq := albsdk.CreateUpdateLoadBalancerAddressTypeConfigRequest()
	updateLbReq.LoadBalancerId = sdkLB.LoadBalancerId
	updateLbReq.AddressType = resLB.Spec.AddressType
	if resLB.Spec.AddressType == util.LoadBalancerAddressTypeInternet &&
		sdkLB.AddressType == util.LoadBalancerAddressTypeIntranet {
		addressTypeConfigZoneMappings := transZoneMappingToUpdateLoadBalancerAddressTypeConfigZoneMappings(resLB.Spec.ZoneMapping)
		updateLbReq.ZoneMappings = &addressTypeConfigZoneMappings
	}
	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("updating loadBalancer address type",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"startTime", startTime,
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		util.Action, util.UpdateALBLoadBalancerAddressType)
	updateLbResp, err := m.auth.ALB.UpdateLoadBalancerAddressTypeConfig(updateLbReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("updated loadBalancer address type",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.UpdateALBLoadBalancerAddressType)

	return nil
}

func filterAlbLoadBalancerResourceTags(tags map[string]string, trackingProvider tracking.TrackingProvider) map[string]string {
	ret := make(map[string]string)
	for k, v := range tags {
		if !trackingProvider.IsAlbIngressTagKey(k) {
			ret[k] = v
		}
	}
	return ret
}

func (m *ALBProvider) updateAlbLoadBalancerTag(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer, trackingProvider tracking.TrackingProvider) error {
	traceID := ctx.Value(util.TraceID)

	if len(resLB.Spec.Tags) == 0 {
		m.logger.V(util.MgrLogLevel).Info("AlbLoadBalancerTag no need update, res tags is nil",
			"sdk", sdkLB.Tags,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		return nil
	}
	var (
		isAlbLoadBalancerTagNeedUpdate = false
	)

	sdkCustomerTagMap := filterAlbLoadBalancerResourceTags(transSDKTagListToMap(sdkLB.Tags), trackingProvider)
	resCustomerTagMap := transTagListToMap(resLB.Spec.Tags)

	if !reflect.DeepEqual(resCustomerTagMap, sdkCustomerTagMap) {
		m.logger.V(util.MgrLogLevel).Info("AlbLoadBalancerTag update",
			"res", resLB.Spec.Tags,
			"sdk", sdkLB.Tags,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		isAlbLoadBalancerTagNeedUpdate = true
	}
	if !isAlbLoadBalancerTagNeedUpdate {
		return nil
	}

	needUnTagMaps := make(map[string]string)
	needTagMaps := make(map[string]string)
	// find needTags map
	for key, resValue := range resCustomerTagMap {
		if sdkValue, exist := sdkCustomerTagMap[key]; exist {
			if sdkValue != resValue {
				needTagMaps[key] = resValue
			}
		} else {
			needTagMaps[key] = resValue
		}
	}

	// find needUnTags map
	for key, sdkValue := range sdkCustomerTagMap {
		if _, exist := resCustomerTagMap[key]; !exist {
			needUnTagMaps[key] = sdkValue
		}
	}
	if len(needUnTagMaps) != 0 {
		if err := m.UnTag(ctx, needUnTagMaps, sdkLB.LoadBalancerId); err != nil {
			m.logger.V(util.MgrLogLevel).Error(err, "unTag load balancer failed",
				"stackID", resLB.Stack().StackID(),
				"resourceID", resLB.ID(),
				"traceID", traceID,
				"loadBalancerID", sdkLB.LoadBalancerId,
				"unTags", needUnTagMaps,
				util.Action, util.UnTagALBResource)
			return err
		}
	}

	if len(needTagMaps) != 0 {
		if err := m.TagWithoutResourceTags(ctx, needTagMaps, sdkLB.LoadBalancerId); err != nil {
			m.logger.V(util.MgrLogLevel).Error(err, "tag load balancer failed",
				"stackID", resLB.Stack().StackID(),
				"resourceID", resLB.ID(),
				"traceID", traceID,
				"loadBalancerID", sdkLB.LoadBalancerId,
				"tags", needTagMaps,
				util.Action, util.TagALBResource)
			return err
		}
	}

	return nil
}

func (m *ALBProvider) UpdateALB(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB albsdk.LoadBalancer, trackingProvider tracking.TrackingProvider) (alb.LoadBalancerStatus, error) {
	if err := m.updateAlbLoadBalancerAttribute(ctx, resLB, &sdkLB); err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	if err := m.updateAlbLoadBalancerDeletionProtection(ctx, resLB, &sdkLB); err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	if err := m.updateAlbLoadBalancerIpv6AddressType(ctx, resLB, &sdkLB); err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	if err := m.updateAlbLoadBalancerAccessLogConfig(ctx, resLB, &sdkLB); err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	if err := m.updateAlbLoadBalancerEdition(ctx, resLB, &sdkLB); err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	if err := m.updateAlbLoadBalancerBandWidthPackage(ctx, resLB, &sdkLB); err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	if err := m.updateAlbLoadBalancerTag(ctx, resLB, &sdkLB, trackingProvider); err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	if err := m.updateAlbLoadBalancerResourceGroup(ctx, resLB, &sdkLB); err != nil {
		return alb.LoadBalancerStatus{}, err
	}
	//if err := m.updateAlbLoadBalancerAddressTypeConfig(ctx, resLB, &sdkLB); err != nil {
	//	return alb.LoadBalancerStatus{}, err
	//}

	return buildResAlbLoadBalancerStatus(sdkLB.LoadBalancerId, sdkLB.DNSName), nil
}

func (m *ALBProvider) ServiceManagedControl(ctx context.Context, resLB *alb.AlbLoadBalancer, lbID string) error {
	traceID := ctx.Value(util.TraceID)

	startTime := time.Now()
	rpcRequest := &requests.RpcRequest{}
	rpcRequest.InitWithApiInfo("Alb", "2020-06-16", "ServiceManagedControl", "alb", "innerAPI")
	region, err := m.auth.Meta.Region()
	if err != nil {
		return err
	}
	resourceUid, err := m.auth.Meta.OwnerAccountID()
	if err != nil {
		return err
	}
	region = reCorrectRegion(region)
	rpcRequest.RegionId = region
	rpcRequest.QueryParams = map[string]string{
		"ServiceManagedMode": "Managed",
		"ResourceType":       "LoadBalancer",
		"ResourceUid":        resourceUid,
		"ResourceIds.1":      lbID,
	}

	m.logger.V(util.MgrLogLevel).Info("service managed control resource",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		"request", rpcRequest,
		util.Action, util.ALBInnerServiceManagedControl)
	response := responses.NewCommonResponse()
	if err := m.DoAction(rpcRequest, response); err != nil {
		return err
	}
	resp := make(map[string]string)
	if err := json.Unmarshal(response.GetHttpContentBytes(), &resp); err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", lbID,
		"requestID", resp["RequestId"],
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.ALBInnerServiceManagedControl)

	return nil
}

func (m *ALBProvider) TagWithoutResourceTags(ctx context.Context, withoutResourceTags map[string]string, lbID string) error {
	traceID := ctx.Value(util.TraceID)

	lbIDs := make([]string, 0)
	lbIDs = append(lbIDs, lbID)
	tags := transTagMapToSDKTagResourcesTagList(withoutResourceTags)
	tagReq := albsdk.CreateTagResourcesRequest()
	tagReq.Tag = &tags
	tagReq.ResourceId = &lbIDs
	tagReq.ResourceType = util.LoadBalancerResourceType
	startTime := time.Now()

	m.logger.V(util.MgrLogLevel).Info("tagging resource without resourceTags",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.TagALBResource)
	tagResp, err := m.auth.ALB.TagResources(tagReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("tagged resource without resourceTags",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"requestID", tagResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.TagALBResource)

	return nil
}

func (m *ALBProvider) Tag(ctx context.Context, resLB *alb.AlbLoadBalancer, lbID string, trackingProvider tracking.TrackingProvider) error {
	traceID := ctx.Value(util.TraceID)

	lbTags := trackingProvider.ResourceTags(resLB.Stack(), resLB, transTagListToMap(resLB.Spec.Tags))
	lbIDs := make([]string, 0)
	lbIDs = append(lbIDs, lbID)
	tags := transTagMapToSDKTagResourcesTagList(lbTags)
	tagReq := albsdk.CreateTagResourcesRequest()
	tagReq.Tag = &tags
	tagReq.ResourceId = &lbIDs
	tagReq.ResourceType = util.LoadBalancerResourceType
	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("tagging resource",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.TagALBResource)
	tagResp, err := m.auth.ALB.TagResources(tagReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("tagged resource",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", lbID,
		"requestID", tagResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.TagALBResource)

	return nil
}
func (m *ALBProvider) UnTag(ctx context.Context, tags map[string]string, lbID string) error {
	traceID := ctx.Value(util.TraceID)

	lbIds := make([]string, 0)
	lbIds = append(lbIds, lbID)
	sdkTags := transTagMapToSDKUnTagResourcesTagList(tags)
	untagReq := albsdk.CreateUnTagResourcesRequest()
	untagReq.Tag = &sdkTags
	untagReq.ResourceId = &lbIds
	untagReq.ResourceType = util.LoadBalancerResourceType
	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("untagging resource",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.UnTagALBResource)
	untagResp, err := m.auth.ALB.UnTagResources(untagReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("untagged resource",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"requestID", untagResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.UnTagALBResource)

	return nil
}
func (m *ALBProvider) preCheckTagConflictForReuse(ctx context.Context, sdkLB *albsdk.GetLoadBalancerAttributeResponse, resLB *alb.AlbLoadBalancer, trackingProvider tracking.TrackingProvider) error {
	if sdkLB.VpcId != resLB.Spec.VpcId {
		return fmt.Errorf("the vpc %s of reused alb %s is not same with cluster vpc %s", sdkLB.VpcId, sdkLB.LoadBalancerId, resLB.Spec.VpcId)
	}

	if len(sdkLB.Tags) == 0 {
		return nil
	}
	sdkTags := transSDKTagListToMap(sdkLB.Tags)
	resTags := trackingProvider.ResourceTags(resLB.Stack(), resLB, transTagListToMap(resLB.Spec.Tags))

	sdkClusterID, okSdkClusterID := sdkTags[trackingProvider.ClusterNameTagKey()]
	resClusterID, okResClusterID := resTags[trackingProvider.ClusterNameTagKey()]
	if okSdkClusterID && okResClusterID {
		if sdkClusterID != resClusterID {
			return fmt.Errorf("alb %s belongs to cluster: %s, cant reuse alb to another cluster: %s",
				sdkLB.LoadBalancerId, sdkClusterID, resClusterID)
		}

		sdkAlbConfig, okSdkAlbConfig := sdkTags[trackingProvider.AlbConfigTagKey()]
		resAlbConfig, okResAlbConfig := resTags[trackingProvider.AlbConfigTagKey()]
		if okSdkAlbConfig && okResAlbConfig {
			if sdkAlbConfig != resAlbConfig {
				return fmt.Errorf("alb %s belongs to albconfig: %s, cant reuse alb to another albconfig: %s",
					sdkLB.LoadBalancerId, sdkAlbConfig, resAlbConfig)
			}
		}
	}

	return nil
}

func (m *ALBProvider) ReuseALB(ctx context.Context, resLB *alb.AlbLoadBalancer, lbID string, trackingProvider tracking.TrackingProvider) (alb.LoadBalancerStatus, error) {
	getLbResp, err := getALBLoadBalancerAttributeFunc(ctx, lbID, m.auth, m.logger)
	if err != nil {
		return alb.LoadBalancerStatus{}, err
	}

	if getLbResp.LoadBalancerEdition == util.LoadBalancerEditionBasic {
		return alb.LoadBalancerStatus{}, fmt.Errorf("LoadBalancer Edition: %s can't use for ingress controller", getLbResp.LoadBalancerEdition)
	}

	if err := m.preCheckTagConflictForReuse(ctx, getLbResp, resLB, trackingProvider); err != nil {
		return alb.LoadBalancerStatus{}, err
	}

	if err = m.Tag(ctx, resLB, lbID, trackingProvider); err != nil {
		return alb.LoadBalancerStatus{}, err
	}

	if resLB.Spec.ForceOverride != nil && *resLB.Spec.ForceOverride {
		loadBalancer := transSDKGetLoadBalancerAttributeResponseToLoadBalancer(*getLbResp)
		return m.UpdateALB(ctx, resLB, loadBalancer, trackingProvider)
	}

	return buildResAlbLoadBalancerStatus(lbID, getLbResp.DNSName), nil
}

func (m *ALBProvider) UnReuseALB(ctx context.Context, lbID string, trackingProvider tracking.TrackingProvider) error {
	getLbResp, err := getALBLoadBalancerAttributeFunc(ctx, lbID, m.auth, m.logger)
	if err != nil {
		return err
	}

	tagsToClean := make(map[string]string, 0)
	for _, tag := range getLbResp.Tags {
		if trackingProvider.IsAlbIngressTagKey(tag.Key) {
			tagsToClean[tag.Key] = tag.Value
		}
	}

	if err = m.UnTag(ctx, tagsToClean, lbID); err != nil {
		return err
	}

	return nil
}
func transSDKGetLoadBalancerAttributeResponseToLoadBalancer(albAttr albsdk.GetLoadBalancerAttributeResponse) albsdk.LoadBalancer {
	return albsdk.LoadBalancer{
		AddressAllocatedMode:         albAttr.AddressAllocatedMode,
		AddressType:                  albAttr.AddressType,
		BandwidthCapacity:            albAttr.BandwidthCapacity,
		BandwidthPackageId:           albAttr.BandwidthPackageId,
		CreateTime:                   albAttr.CreateTime,
		DNSName:                      albAttr.DNSName,
		ServiceManagedEnabled:        albAttr.ServiceManagedEnabled,
		ServiceManagedMode:           albAttr.ServiceManagedMode,
		LoadBalancerBussinessStatus:  albAttr.LoadBalancerBussinessStatus,
		LoadBalancerEdition:          albAttr.LoadBalancerEdition,
		LoadBalancerId:               albAttr.LoadBalancerId,
		LoadBalancerName:             albAttr.LoadBalancerName,
		LoadBalancerStatus:           albAttr.LoadBalancerStatus,
		ResourceGroupId:              albAttr.ResourceGroupId,
		VpcId:                        albAttr.VpcId,
		AccessLogConfig:              albAttr.AccessLogConfig,
		DeletionProtectionConfig:     albAttr.DeletionProtectionConfig,
		LoadBalancerBillingConfig:    albAttr.LoadBalancerBillingConfig,
		ModificationProtectionConfig: albAttr.ModificationProtectionConfig,
		LoadBalancerOperationLocks:   albAttr.LoadBalancerOperationLocks,
		Tags:                         albAttr.Tags,
	}
}

var getALBLoadBalancerAttributeFunc = func(ctx context.Context, lbID string, auth *base.ClientMgr, logger logr.Logger) (*albsdk.GetLoadBalancerAttributeResponse, error) {
	traceID := ctx.Value(util.TraceID)

	getLbReq := albsdk.CreateGetLoadBalancerAttributeRequest()
	getLbReq.LoadBalancerId = lbID
	startTime := time.Now()
	logger.V(util.MgrLogLevel).Info("getting loadBalancer attribute",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.GetALBLoadBalancerAttribute)
	getLbResp, err := auth.ALB.GetLoadBalancerAttribute(getLbReq)
	if err != nil {
		return nil, err
	}
	logger.V(util.MgrLogLevel).Info("got loadBalancer attribute",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"loadBalancerStatus", getLbResp.LoadBalancerStatus,
		"requestID", getLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.GetALBLoadBalancerAttribute)
	return getLbResp, nil
}

var disableALBDeletionProtectionFunc = func(ctx context.Context, lbID string, auth *base.ClientMgr, logger logr.Logger) (*albsdk.DisableDeletionProtectionResponse, error) {
	traceID := ctx.Value(util.TraceID)

	updateLbReq := albsdk.CreateDisableDeletionProtectionRequest()
	updateLbReq.ResourceId = lbID
	startTime := time.Now()
	logger.V(util.MgrLogLevel).Info("disabling delete protection",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.DisableALBDeletionProtection)
	updateLbResp, err := auth.ALB.DisableDeletionProtection(updateLbReq)
	if err != nil {
		return nil, err
	}
	logger.V(util.MgrLogLevel).Info("disabled delete protection",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.DisableALBDeletionProtection)

	return updateLbResp, nil
}

var enableALBDeletionProtectionFunc = func(ctx context.Context, lbID string, auth *base.ClientMgr, logger logr.Logger) (*albsdk.EnableDeletionProtectionResponse, error) {
	traceID := ctx.Value(util.TraceID)

	updateLbReq := albsdk.CreateEnableDeletionProtectionRequest()
	updateLbReq.ResourceId = lbID
	startTime := time.Now()
	logger.V(util.MgrLogLevel).Info("enabling delete protection",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.EnableALBDeletionProtection)
	updateLbResp, err := auth.ALB.EnableDeletionProtection(updateLbReq)
	if err != nil {
		return nil, err
	}
	logger.V(util.MgrLogLevel).Info("enabled delete protection",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.EnableALBDeletionProtection)

	return updateLbResp, nil
}

var disableLoadBalancerIpv6InternetFunc = func(ctx context.Context, lbID string, auth *base.ClientMgr, logger logr.Logger) (*albsdk.DisableLoadBalancerIpv6InternetResponse, error) {
	traceID := ctx.Value(util.TraceID)
	updateLbReq := albsdk.CreateDisableLoadBalancerIpv6InternetRequest()
	updateLbReq.LoadBalancerId = lbID
	startTime := time.Now()
	logger.V(util.MgrLogLevel).Info("disabling internet ipv6 address type",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.DisableALBIpv6Internet)
	updateLbResp, err := auth.ALB.DisableLoadBalancerIpv6Internet(updateLbReq)
	if err != nil {
		return nil, err
	}
	logger.V(util.MgrLogLevel).Info("disabling internet ipv6 address type",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.DisableALBIpv6Internet)
	return updateLbResp, nil
}

var enableLoadBalancerIpv6InternetFunc = func(ctx context.Context, lbID string, auth *base.ClientMgr, logger logr.Logger) (*albsdk.EnableLoadBalancerIpv6InternetResponse, error) {
	traceID := ctx.Value(util.TraceID)
	updateLbReq := albsdk.CreateEnableLoadBalancerIpv6InternetRequest()
	updateLbReq.LoadBalancerId = lbID
	startTime := time.Now()
	logger.V(util.MgrLogLevel).Info("enabling internet ipv6 address type",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.EnableALBIpv6Internet)
	updateLbResp, err := auth.ALB.EnableLoadBalancerIpv6Internet(updateLbReq)
	if err != nil {
		return nil, err
	}
	logger.V(util.MgrLogLevel).Info("enabling internet ipv6 address type",
		"traceID", traceID,
		"loadBalancerID", lbID,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.EnableALBIpv6Internet)
	return updateLbResp, nil
}

var deleteALBLoadBalancerFunc = func(ctx context.Context, m *ALBProvider, lbID string) (*albsdk.DeleteLoadBalancerResponse, error) {
	traceID := ctx.Value(util.TraceID)

	lbReq := albsdk.CreateDeleteLoadBalancerRequest()
	lbReq.LoadBalancerId = lbID

	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("deleting loadBalancer",
		"loadBalancerID", lbID,
		"traceID", traceID,
		"startTime", startTime,
		util.Action, util.DeleteALBLoadBalancer)
	lsResp, err := m.auth.ALB.DeleteLoadBalancer(lbReq)
	if err != nil {
		return nil, err
	}
	m.logger.V(util.MgrLogLevel).Info("deleted loadBalancer",
		"loadBalancerID", lbID,
		"traceID", traceID,
		"requestID", lsResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.DeleteALBLoadBalancer)

	return lsResp, nil
}

func (m *ALBProvider) DeleteALB(ctx context.Context, lbID string) error {
	getLbResp, err := getALBLoadBalancerAttributeFunc(ctx, lbID, m.auth, m.logger)
	if err != nil {
		return err
	}

	if getLbResp.DeletionProtectionConfig.Enabled {
		_, err := disableALBDeletionProtectionFunc(ctx, lbID, m.auth, m.logger)
		if err != nil {
			return err
		}
	}

	if _, err = deleteALBLoadBalancerFunc(ctx, m, lbID); err != nil {
		return err
	}

	return nil
}

func transSDKModificationProtectionConfigToCreateLb(mpc alb.ModificationProtectionConfig) albsdk.CreateLoadBalancerModificationProtectionConfig {
	return albsdk.CreateLoadBalancerModificationProtectionConfig{
		Reason: mpc.Reason,
		Status: mpc.Status,
	}
}

func transSDKLoadBalancerBillingConfigToCreateLb(lbc alb.LoadBalancerBillingConfig) albsdk.CreateLoadBalancerLoadBalancerBillingConfig {
	return albsdk.CreateLoadBalancerLoadBalancerBillingConfig{
		PayType:            lbc.PayType,
		BandwidthPackageId: lbc.BandWidthPackageId,
	}
}

func transSDKZoneMappingsToCreateLb(zoneMappings []alb.ZoneMapping) *[]albsdk.CreateLoadBalancerZoneMappings {
	createLbZoneMappings := make([]albsdk.CreateLoadBalancerZoneMappings, 0)

	for _, zoneMapping := range zoneMappings {
		createLbZoneMapping := albsdk.CreateLoadBalancerZoneMappings{
			VSwitchId: zoneMapping.VSwitchId,
			ZoneId:    zoneMapping.ZoneId,
		}
		createLbZoneMappings = append(createLbZoneMappings, createLbZoneMapping)
	}

	return &createLbZoneMappings
}

func buildSDKCreateAlbLoadBalancerRequest(lbSpec alb.ALBLoadBalancerSpec) (*albsdk.CreateLoadBalancerRequest, error) {
	createLbReq := albsdk.CreateCreateLoadBalancerRequest()
	if len(lbSpec.VpcId) == 0 {
		return nil, fmt.Errorf("invalid load balancer vpc id: %s", lbSpec.VpcId)
	}
	createLbReq.VpcId = lbSpec.VpcId

	if !isAlbLoadBalancerAddressTypeValid(lbSpec.AddressType) {
		return nil, fmt.Errorf("invalid load balancer address type: %s", lbSpec.AddressType)
	}
	createLbReq.AddressType = lbSpec.AddressType

	if !isAlbLoadBalancerAddressIpVersionValid(lbSpec.AddressIpVersion) {
		return nil, fmt.Errorf("invalid load balancer address ip version: %s", lbSpec.AddressIpVersion)
	}
	createLbReq.AddressIpVersion = lbSpec.AddressIpVersion

	createLbReq.LoadBalancerName = lbSpec.LoadBalancerName

	createLbReq.DeletionProtectionEnabled = requests.NewBoolean(lbSpec.DeletionProtectionConfig.Enabled)

	if !isAlbLoadBalancerModificationProtectionStatusValid(lbSpec.ModificationProtectionConfig.Status) {
		return nil, fmt.Errorf("invalid load balancer modification protection config: %v", lbSpec.ModificationProtectionConfig)
	}
	createLbReq.ModificationProtectionConfig = transSDKModificationProtectionConfigToCreateLb(lbSpec.ModificationProtectionConfig)

	if len(lbSpec.ZoneMapping) == 0 {
		return nil, fmt.Errorf("empty load balancer zone mapping")
	}
	createLbReq.ZoneMappings = transSDKZoneMappingsToCreateLb(lbSpec.ZoneMapping)

	if !isLoadBalancerAddressAllocatedModeValid(lbSpec.AddressAllocatedMode) {
		return nil, fmt.Errorf("invalid load balancer address allocate mode: %s", lbSpec.AddressAllocatedMode)
	}
	createLbReq.AddressAllocatedMode = lbSpec.AddressAllocatedMode

	createLbReq.ResourceGroupId = lbSpec.ResourceGroupId

	if !isAlbLoadBalancerEditionValid(lbSpec.LoadBalancerEdition) {
		return nil, fmt.Errorf("invalid load balancer edition: %s", lbSpec.LoadBalancerEdition)
	}
	createLbReq.LoadBalancerEdition = lbSpec.LoadBalancerEdition

	if !isAlbLoadBalancerLoadBalancerPayTypeValid(lbSpec.LoadBalancerBillingConfig.PayType) {
		return nil, fmt.Errorf("invalid load balancer paytype: %s", lbSpec.LoadBalancerBillingConfig.PayType)
	}
	createLbReq.LoadBalancerBillingConfig = transSDKLoadBalancerBillingConfigToCreateLb(lbSpec.LoadBalancerBillingConfig)

	return createLbReq, nil
}

func transSDKModificationProtectionConfigToUpdateLb(mpc alb.ModificationProtectionConfig) albsdk.UpdateLoadBalancerAttributeModificationProtectionConfig {
	return albsdk.UpdateLoadBalancerAttributeModificationProtectionConfig{
		Reason: mpc.Reason,
		Status: mpc.Status,
	}
}

func (m *ALBProvider) updateAlbLoadBalancerAttribute(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)

	var (
		isModificationProtectionConfigModifyNeedUpdate = false
		isLoadBalancerNameNeedUpdate                   = false
	)

	if !isAlbLoadBalancerModificationProtectionStatusValid(resLB.Spec.ModificationProtectionConfig.Status) {
		return fmt.Errorf("invalid load balancer modification protection config: %v", resLB.Spec.ModificationProtectionConfig)
	}
	modificationProtectionConfig := transModificationProtectionConfigToSDK(resLB.Spec.ModificationProtectionConfig)
	if modificationProtectionConfig != sdkLB.ModificationProtectionConfig {
		m.logger.V(util.MgrLogLevel).Info("ModificationProtectionConfig update",
			"res", resLB.Spec.ModificationProtectionConfig,
			"sdk", sdkLB.ModificationProtectionConfig,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		isModificationProtectionConfigModifyNeedUpdate = true
	}
	if resLB.Spec.LoadBalancerName != sdkLB.LoadBalancerName {
		m.logger.V(util.MgrLogLevel).Info("LoadBalancerName update",
			"res", resLB.Spec.LoadBalancerName,
			"sdk", sdkLB.LoadBalancerName,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		isLoadBalancerNameNeedUpdate = true
	}

	if !isLoadBalancerNameNeedUpdate && !isModificationProtectionConfigModifyNeedUpdate {
		return nil
	}

	updateLbReq := albsdk.CreateUpdateLoadBalancerAttributeRequest()
	updateLbReq.LoadBalancerId = sdkLB.LoadBalancerId
	if isModificationProtectionConfigModifyNeedUpdate {
		updateLbReq.ModificationProtectionConfig = transSDKModificationProtectionConfigToUpdateLb(resLB.Spec.ModificationProtectionConfig)
	}
	if isLoadBalancerNameNeedUpdate {
		updateLbReq.LoadBalancerName = resLB.Spec.LoadBalancerName
	}

	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("updating loadBalancer attribute",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		"startTime", startTime,
		util.Action, util.UpdateALBLoadBalancerAttribute)
	updateLbResp, err := m.auth.ALB.UpdateLoadBalancerAttribute(updateLbReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("updating loadBalancer attribute",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.UpdateALBLoadBalancerAttribute)
	if _, err = m.waitAlbLoadBalancerAttributeStatus(ctx, updateLbReq.LoadBalancerId); err != nil {
		return err
	}
	return nil
}

func (m *ALBProvider) waitAlbLoadBalancerAttributeStatus(ctx context.Context, loadBalancerId string) (*albsdk.GetLoadBalancerAttributeResponse, error) {
	var getLbResp *albsdk.GetLoadBalancerAttributeResponse
	var err error
	for i := 0; i < util.UpdateLoadBalancerAttributeWaitActiveMaxRetryTimes; i++ {
		getLbResp, err = getALBLoadBalancerAttributeFunc(ctx, loadBalancerId, m.auth, m.logger)
		if err != nil {
			return getLbResp, err
		}
		if isAlbLoadBalancerActive(getLbResp.LoadBalancerStatus) {
			break
		}
		time.Sleep(util.UpdateLoadBalancerAttributeWaitActiveRetryInterval)
	}
	return getLbResp, nil
}

func (m *ALBProvider) updateAlbLoadBalancerIpv6AddressType(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)
	var (
		isIpv6AddressTypeNeedUpdate = false
	)
	if sdkLB.Ipv6AddressType != "" && resLB.Spec.Ipv6AddressType != sdkLB.Ipv6AddressType {
		m.logger.V(util.MgrLogLevel).Info("Ipv6AddressType update",
			"res", resLB.Spec.Ipv6AddressType,
			"sdk", sdkLB.Ipv6AddressType,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		isIpv6AddressTypeNeedUpdate = true
	}
	if !isIpv6AddressTypeNeedUpdate {
		return nil
	}
	if util.LoadBalancerIpv6AddressTypeInternet == resLB.Spec.Ipv6AddressType {
		_, err := enableLoadBalancerIpv6InternetFunc(ctx, sdkLB.LoadBalancerId, m.auth, m.logger)
		if err != nil {
			return err
		}
	} else if util.LoadBalancerIpv6AddressTypeIntranet == resLB.Spec.Ipv6AddressType {
		_, err := disableLoadBalancerIpv6InternetFunc(ctx, sdkLB.LoadBalancerId, m.auth, m.logger)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ALBProvider) updateAlbLoadBalancerDeletionProtection(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)

	var (
		isDeletionProtectionNeedUpdate = false
	)

	if resLB.Spec.DeletionProtectionConfig.Enabled != sdkLB.DeletionProtectionConfig.Enabled {
		m.logger.V(util.MgrLogLevel).Info("DeletionProtectionConfig update",
			"res", resLB.Spec.DeletionProtectionConfig.Enabled,
			"sdk", sdkLB.DeletionProtectionConfig.Enabled,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		isDeletionProtectionNeedUpdate = true
	}
	if !isDeletionProtectionNeedUpdate {
		return nil
	}

	if resLB.Spec.DeletionProtectionConfig.Enabled && !sdkLB.DeletionProtectionConfig.Enabled {
		_, err := enableALBDeletionProtectionFunc(ctx, sdkLB.LoadBalancerId, m.auth, m.logger)
		if err != nil {
			return err
		}
	} else if !resLB.Spec.DeletionProtectionConfig.Enabled && sdkLB.DeletionProtectionConfig.Enabled {
		_, err := disableALBDeletionProtectionFunc(ctx, sdkLB.LoadBalancerId, m.auth, m.logger)
		if err != nil {
			return err
		}
	}

	return nil
}

func reCorrectRegion(region string) string {
	switch region {
	case "cn-shenzhen-finance-1":
		return "cn-shenzhen-finance"
	}
	return region
}

func (m *ALBProvider) DissociateAccessLogFromALB(ctx context.Context, lbID string, resLB *alb.AlbLoadBalancer) error {
	traceID := ctx.Value(util.TraceID)

	rpcRequest := &requests.RpcRequest{}
	rpcRequest.InitWithApiInfo("Sls", "2019-10-23", "CloseProductDataCollection", "sls", "innerAPI")
	region, err := m.auth.Meta.Region()
	if err != nil {
		return err
	}
	region = reCorrectRegion(region)
	rpcRequest.Domain = fmt.Sprintf("%s%s", region, util.DefaultLogDomainSuffix)
	rpcRequest.QueryParams = map[string]string{
		"DataType":       "alb.access_log",
		"InstanceRegion": region,
		"InstanceId":     lbID,
	}
	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("close product data collection",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"request", rpcRequest,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.CloseProductDataCollection)
	response := responses.NewCommonResponse()
	err = m.auth.SLS.DoAction(rpcRequest, response)
	if err != nil {
		return err
	}
	if !response.IsSuccess() {
		err := fmt.Errorf("failed close SLS product data, reponse: %v", response.GetHttpContentString())
		m.logger.V(util.MgrLogLevel).Info("close product data collection",
			"stackID", resLB.Stack().StackID(),
			"resourceID", resLB.ID(),
			"loadBalancerID", lbID,
			"traceID", traceID,
			"error", err.Error(),
			util.Action, util.CloseProductDataCollection)
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("close product data collection",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"loadBalancerID", lbID,
		"traceID", traceID,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.CloseProductDataCollection)
	return nil
}

func (m *ALBProvider) AnalyzeAndAssociateAccessLogToALB(ctx context.Context, lbID string, resLB *alb.AlbLoadBalancer) error {
	traceID := ctx.Value(util.TraceID)

	logProject := resLB.Spec.AccessLogConfig.LogProject
	logStore := resLB.Spec.AccessLogConfig.LogStore

	if !isLogProjectNameValid(logProject) || !isLogStoreNameValid(logStore) {
		return fmt.Errorf("invalid name of logProject: %s or logStore: %s", logProject, logStore)
	}

	rpcRequest := &requests.RpcRequest{}
	rpcRequest.InitWithApiInfo("Sls", "2019-10-23", "OpenProductDataCollection", "sls", "innerAPI")
	region, err := m.auth.Meta.Region()
	if err != nil {
		return err
	}
	region = reCorrectRegion(region)
	rpcRequest.Domain = fmt.Sprintf("%s%s", region, util.DefaultLogDomainSuffix)
	rpcRequest.QueryParams = map[string]string{
		"DataType":        "alb.access_log",
		"InstanceRegion":  region,
		"InstanceId":      lbID,
		"TargetSLSRegion": region,
		"TargetProject":   logProject,
		"TargetStore":     logStore,
	}
	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("open product data collection",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"request", rpcRequest,
		"loadBalancerID", lbID,
		"startTime", startTime,
		util.Action, util.OpenProductDataCollection)
	response := responses.NewCommonResponse()
	err = m.auth.SLS.DoAction(rpcRequest, response)
	if err != nil {
		return err
	}
	if !response.IsSuccess() {
		err := fmt.Errorf("failed open SLS product data, reponse: %v", response.GetHttpContentString())
		m.logger.V(util.MgrLogLevel).Info("open product data collection",
			"stackID", resLB.Stack().StackID(),
			"resourceID", resLB.ID(),
			"loadBalancerID", lbID,
			"traceID", traceID,
			"error", err.Error(),
			util.Action, util.OpenProductDataCollection)
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("open product data collection",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"loadBalancerID", lbID,
		"traceID", traceID,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.OpenProductDataCollection)
	return nil
}

func (m *ALBProvider) updateAlbLoadBalancerAccessLogConfig(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)

	var (
		isAccessLogConfigNeedUpdate = false
	)
	accessLogConfig := transAccessLogConfigToSDK(resLB.Spec.AccessLogConfig)
	if accessLogConfig != sdkLB.AccessLogConfig {
		m.logger.Info("LoadBalancer AccessLogConfig update",
			"res", resLB.Spec.AccessLogConfig,
			"sdk", sdkLB.AccessLogConfig,
			"traceID", traceID)
		isAccessLogConfigNeedUpdate = true
	}
	if !isAccessLogConfigNeedUpdate {
		return nil
	}

	if len(sdkLB.AccessLogConfig.LogProject) != 0 && len(sdkLB.AccessLogConfig.LogStore) != 0 {
		if err := m.DissociateAccessLogFromALB(ctx, sdkLB.LoadBalancerId, resLB); err != nil {
			return err
		}
	}

	if len(resLB.Spec.AccessLogConfig.LogProject) != 0 && len(resLB.Spec.AccessLogConfig.LogStore) != 0 {
		if err := m.AnalyzeAndAssociateAccessLogToALB(ctx, sdkLB.LoadBalancerId, resLB); err != nil {
			return err
		}
	}
	return nil
}

func (m *ALBProvider) updateAlbLoadBalancerBandWidthPackage(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)
	var (
		isLoadBalancerBandWidthPackageNeedUpdate = false
	)

	if !strings.EqualFold(resLB.Spec.LoadBalancerBillingConfig.BandWidthPackageId, sdkLB.BandwidthPackageId) {
		m.logger.V(util.MgrLogLevel).Info("LoadBalancer BandwidthPackageId update",
			"res", resLB.Spec.LoadBalancerBillingConfig.BandWidthPackageId,
			"sdk", sdkLB.BandwidthPackageId,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		isLoadBalancerBandWidthPackageNeedUpdate = true
	}
	if !isLoadBalancerBandWidthPackageNeedUpdate {
		return nil
	}

	if isLoadBalancerBandWidthPackageNeedUpdate &&
		len(resLB.Spec.LoadBalancerBillingConfig.BandWidthPackageId) != 0 &&
		len(sdkLB.BandwidthPackageId) == 0 {
		if resLB.Spec.AddressType != util.LoadBalancerAddressTypeInternet {
			return fmt.Errorf("intranet addressType cannot set common bandwidth package")
		}
		if err := m.attachCommonBandwidthPackageToALB(ctx, resLB, sdkLB); err != nil {
			return err
		}
	}

	if isLoadBalancerBandWidthPackageNeedUpdate &&
		len(sdkLB.BandwidthPackageId) != 0 &&
		len(resLB.Spec.LoadBalancerBillingConfig.BandWidthPackageId) != 0 {
		if resLB.Spec.AddressType != util.LoadBalancerAddressTypeInternet {
			return fmt.Errorf("intranet addressType cannot set common bandwidth package")
		}
		if err := m.detachCommonBandwidthPackageFromALB(ctx, resLB, sdkLB); err != nil {
			return err
		}
		if err := m.attachCommonBandwidthPackageToALB(ctx, resLB, sdkLB); err != nil {
			return err
		}
	}

	return nil
}

func (m *ALBProvider) attachCommonBandwidthPackageToALB(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)

	bandWidthPackageId := resLB.Spec.LoadBalancerBillingConfig.BandWidthPackageId
	getLbResp, err := getALBLoadBalancerAttributeFunc(ctx, sdkLB.LoadBalancerId, m.auth, m.logger)
	if err != nil {
		return err
	}
	updateLbReq := albsdk.CreateAttachCommonBandwidthPackageToLoadBalancerRequest()
	updateLbReq.LoadBalancerId = sdkLB.LoadBalancerId
	updateLbReq.RegionId = getLbResp.RegionId
	updateLbReq.BandwidthPackageId = bandWidthPackageId

	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("attaching loadBalancer common bandwidth package",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"startTime", startTime,
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		util.Action, util.AttachCommonBandwidthPackageToALBLoadBalancer)
	updateLbResp, err := m.auth.ALB.AttachCommonBandwidthPackageToLoadBalancer(updateLbReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("attached loadBalancer common bandwidth package",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.AttachCommonBandwidthPackageToALBLoadBalancer)
	if _, err = m.waitAlbLoadBalancerAttributeStatus(ctx, updateLbReq.LoadBalancerId); err != nil {
		return err
	}
	return nil
}

func (m *ALBProvider) detachCommonBandwidthPackageFromALB(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)

	bandWidthPackageId := sdkLB.BandwidthPackageId
	getLbResp, err := getALBLoadBalancerAttributeFunc(ctx, sdkLB.LoadBalancerId, m.auth, m.logger)
	if err != nil {
		return err
	}
	updateLbReq := albsdk.CreateDetachCommonBandwidthPackageFromLoadBalancerRequest()
	updateLbReq.LoadBalancerId = sdkLB.LoadBalancerId
	updateLbReq.RegionId = getLbResp.RegionId
	updateLbReq.BandwidthPackageId = bandWidthPackageId

	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("detaching loadBalancer common bandwidth package",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"startTime", startTime,
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		util.Action, util.DetachCommonBandwidthPackageFromALBLoadBalancer)
	updateLbResp, err := m.auth.ALB.DetachCommonBandwidthPackageFromLoadBalancer(updateLbReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("detached loadBalancer common bandwidth package",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.DetachCommonBandwidthPackageFromALBLoadBalancer)
	if _, err = m.waitAlbLoadBalancerAttributeStatus(ctx, updateLbReq.LoadBalancerId); err != nil {
		return err
	}
	return nil
}

func (m *ALBProvider) updateAlbLoadBalancerEdition(ctx context.Context, resLB *alb.AlbLoadBalancer, sdkLB *albsdk.LoadBalancer) error {
	traceID := ctx.Value(util.TraceID)
	// do not update edition when reusing alb
	if len(resLB.Spec.LoadBalancerId) != 0 {
		return nil
	}
	var (
		isLoadBalancerEditionNeedUpdate = false
	)

	if !isAlbLoadBalancerEditionValid(resLB.Spec.LoadBalancerEdition) {
		return fmt.Errorf("invalid load balancer edition: %s", resLB.Spec.LoadBalancerEdition)
	}

	if sdkLB.LoadBalancerEdition != util.LoadBalancerEditionBasic &&
		resLB.Spec.LoadBalancerEdition == util.LoadBalancerEditionBasic {
		return fmt.Errorf("downgrade not allowed for alb from %s to %s", sdkLB.LoadBalancerEdition, resLB.Spec.LoadBalancerEdition)
	}
	if !strings.EqualFold(resLB.Spec.LoadBalancerEdition, sdkLB.LoadBalancerEdition) {
		m.logger.V(util.MgrLogLevel).Info("LoadBalancer Edition update",
			"res", resLB.Spec.LoadBalancerEdition,
			"sdk", sdkLB.LoadBalancerEdition,
			"loadBalancerID", sdkLB.LoadBalancerId,
			"traceID", traceID)
		isLoadBalancerEditionNeedUpdate = true
	}
	if !isLoadBalancerEditionNeedUpdate {
		return nil
	}

	updateLbReq := albsdk.CreateUpdateLoadBalancerEditionRequest()
	updateLbReq.LoadBalancerId = sdkLB.LoadBalancerId
	updateLbReq.LoadBalancerEdition = resLB.Spec.LoadBalancerEdition
	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("updating loadBalancer edition",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"startTime", startTime,
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		util.Action, util.UpdateALBLoadBalancerEdition)
	updateLbResp, err := m.auth.ALB.UpdateLoadBalancerEdition(updateLbReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("updated loadBalancer edition",
		"stackID", resLB.Stack().StackID(),
		"resourceID", resLB.ID(),
		"traceID", traceID,
		"loadBalancerID", sdkLB.LoadBalancerId,
		"requestID", updateLbResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.UpdateALBLoadBalancerEdition)
	if _, err = m.waitAlbLoadBalancerAttributeStatus(ctx, updateLbReq.LoadBalancerId); err != nil {
		return err
	}
	return nil
}

func transTagMapToSDKTagResourcesTagList(tagMap map[string]string) []albsdk.TagResourcesTag {
	tagList := make([]albsdk.TagResourcesTag, 0)
	for k, v := range tagMap {
		tagList = append(tagList, albsdk.TagResourcesTag{
			Key:   k,
			Value: v,
		})
	}
	return tagList
}
func transTagMapToSDKUnTagResourcesTagList(tagMap map[string]string) []albsdk.UnTagResourcesTag {
	tagList := make([]albsdk.UnTagResourcesTag, 0)
	for k, v := range tagMap {
		tagList = append(tagList, albsdk.UnTagResourcesTag{
			Key:   k,
			Value: v,
		})
	}
	return tagList
}
func transTagListToMap(tagList []alb.ALBTag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tagList {
		tagMap[tag.Key] = tag.Value
	}
	return tagMap
}

func transSDKTagListToMap(tagList []albsdk.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tagList {
		tagMap[tag.Key] = tag.Value
	}
	return tagMap
}
func buildResAlbLoadBalancerStatus(lbID, DNSName string) alb.LoadBalancerStatus {
	return alb.LoadBalancerStatus{
		LoadBalancerID: lbID,
		DNSName:        DNSName,
	}
}

func isAlbLoadBalancerAddressTypeValid(addressType string) bool {
	if strings.EqualFold(addressType, util.LoadBalancerAddressTypeInternet) ||
		strings.EqualFold(addressType, util.LoadBalancerAddressTypeIntranet) {
		return true
	}
	return false
}

func isAlbLoadBalancerModificationProtectionStatusValid(modificationProtectionStatus string) bool {
	if strings.EqualFold(modificationProtectionStatus, util.LoadBalancerModificationProtectionStatusNonProtection) ||
		strings.EqualFold(modificationProtectionStatus, util.LoadBalancerModificationProtectionStatusConsoleProtection) {
		return true
	}
	return false
}

func isLoadBalancerAddressAllocatedModeValid(addressAllocatedMode string) bool {
	if strings.EqualFold(addressAllocatedMode, util.LoadBalancerAddressAllocatedModeFixed) ||
		strings.EqualFold(addressAllocatedMode, util.LoadBalancerAddressAllocatedModeDynamic) {
		return true
	}
	return false
}

func isAlbLoadBalancerAddressIpVersionValid(addressIpVersion string) bool {
	if strings.EqualFold(addressIpVersion, util.LoadBalancerAddressIpVersionIPv4) ||
		strings.EqualFold(addressIpVersion, util.LoadBalancerAddressIpVersionDualStack) {
		return true
	}
	return false
}

func (p ALBProvider) TagALBResources(request *albsdk.TagResourcesRequest) (response *albsdk.TagResourcesResponse, err error) {
	return p.auth.ALB.TagResources(request)
}
func (p ALBProvider) UnTagALBResources(request *albsdk.UnTagResourcesRequest) (response *albsdk.UnTagResourcesResponse, err error) {
	return p.auth.ALB.UnTagResources(request)
}
func (p ALBProvider) DescribeALBZones(request *albsdk.DescribeZonesRequest) (response *albsdk.DescribeZonesResponse, err error) {
	return p.auth.ALB.DescribeZones(request)
}

func isAlbLoadBalancerEditionValid(edition string) bool {
	if strings.EqualFold(edition, util.LoadBalancerEditionBasic) ||
		strings.EqualFold(edition, util.LoadBalancerEditionStandard) ||
		strings.EqualFold(edition, util.LoadBalancerEditionWaf) {
		return true
	}
	return false
}

func isAlbLoadBalancerLoadBalancerPayTypeValid(payType string) bool {
	return strings.EqualFold(payType, util.LoadBalancerPayTypePostPay)
}

func isLogProjectNameValid(logProject string) bool {
	if len(logProject) < util.MinLogProjectNameLen || len(logProject) > util.MaxLogProjectNameLen {
		return false
	}
	return true
}

func isLogStoreNameValid(logStore string) bool {
	if len(logStore) < util.MinLogStoreNameLen || len(logStore) > util.MaxLogStoreNameLen {
		return false
	}
	return true
}
func transAccessLogConfigToSDK(a alb.AccessLogConfig) albsdk.AccessLogConfig {
	return albsdk.AccessLogConfig{
		LogProject: a.LogProject,
		LogStore:   a.LogStore,
	}
}
func transModificationProtectionConfigToSDK(m alb.ModificationProtectionConfig) albsdk.ModificationProtectionConfig {
	return albsdk.ModificationProtectionConfig{
		Reason: m.Reason,
		Status: m.Status,
	}
}

func transZoneMappingToUpdateLoadBalancerAddressTypeConfigZoneMappings(m []alb.ZoneMapping) []albsdk.UpdateLoadBalancerAddressTypeConfigZoneMappings {
	updateZoneMapping := make([]albsdk.UpdateLoadBalancerAddressTypeConfigZoneMappings, 0)
	for _, zoneMapping := range m {
		if zoneMapping.AllocationId != "" {
			updateZoneMapping = append(updateZoneMapping, albsdk.UpdateLoadBalancerAddressTypeConfigZoneMappings{
				VSwitchId:    zoneMapping.VSwitchId,
				ZoneId:       zoneMapping.ZoneId,
				AllocationId: zoneMapping.AllocationId,
			})
		}
	}
	return updateZoneMapping
}
