package albconfigmanager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	v1 "k8s.io/alibaba-load-balancer-controller/pkg/apis/alibabacloud/v1"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
)

const (
	ApplicationLoadBalancerResource = "ApplicationLoadBalancer"
)

func (t *defaultModelBuildTask) buildAlbLoadBalancer(ctx context.Context, albconfig *v1.AlbConfig) (*alb.AlbLoadBalancer, error) {
	lbSpec, err := t.buildAlbLoadBalancerSpec(ctx, albconfig)
	if err != nil {
		return nil, err
	}
	lb := alb.NewAlbLoadBalancer(t.stack, ApplicationLoadBalancerResource, lbSpec)
	t.loadBalancer = lb
	if t.albconfig.Status.LoadBalancer.DNSName != "" && t.albconfig.Status.LoadBalancer.Id != "" {
		lbStatus := t.buildAlbLoadBalancerStatus()
		lb.SetStatus(lbStatus)
	}
	return lb, nil
}

func (t *defaultModelBuildTask) buildAlbLoadBalancerSpec(ctx context.Context, albConfig *v1.AlbConfig) (alb.ALBLoadBalancerSpec, error) {
	lbModel := alb.ALBLoadBalancerSpec{}
	lbModel.LoadBalancerId = albConfig.Spec.LoadBalancer.Id
	lbModel.ForceOverride = albConfig.Spec.LoadBalancer.ForceOverride
	forceOverride := false
	if lbModel.ForceOverride == nil {
		lbModel.ForceOverride = &forceOverride
	}
	lbModel.ListenerForceOverride = albConfig.Spec.LoadBalancer.ListenerForceOverride
	listenerForceOverride := false
	if lbModel.ListenerForceOverride == nil {
		lbModel.ListenerForceOverride = &listenerForceOverride
	}
	if len(albConfig.Spec.LoadBalancer.Name) != 0 {
		lbModel.LoadBalancerName = albConfig.Spec.LoadBalancer.Name
	} else {
		lbName, err := t.buildAlbLoadBalancerName()
		if err != nil {
			return alb.ALBLoadBalancerSpec{}, nil
		}
		lbModel.LoadBalancerName = lbName
	}
	lbModel.VpcId = t.vpcID
	lbModel.AccessLogConfig = alb.AccessLogConfig{
		LogStore:   albConfig.Spec.LoadBalancer.AccessLogConfig.LogStore,
		LogProject: albConfig.Spec.LoadBalancer.AccessLogConfig.LogProject,
	}

	zoneMappings := make([]alb.ZoneMapping, 0)
	if albConfig.Spec.LoadBalancer.Id == "" {
		if len(albConfig.Spec.LoadBalancer.ZoneMappings) != 0 {
			vSwitchIds := make([]string, 0)
			vSwitchAllocationID := make(map[string]string, 0)
			for _, zm := range albConfig.Spec.LoadBalancer.ZoneMappings {
				vSwitchIds = append(vSwitchIds, zm.VSwitchId)
				if zm.AllocationId != "" {
					vSwitchAllocationID[zm.VSwitchId] = zm.AllocationId
				}
			}
			vSwitches, err := t.vSwitchResolver.ResolveViaIDSlice(ctx, vSwitchIds)
			if err != nil {
				return alb.ALBLoadBalancerSpec{}, err
			}
			for _, vSwitch := range vSwitches {
				if _, ok := vSwitchAllocationID[vSwitch.VSwitchId]; ok {
					zoneMappings = append(zoneMappings, alb.ZoneMapping{
						VSwitchId:    vSwitch.VSwitchId,
						ZoneId:       vSwitch.ZoneId,
						AllocationId: vSwitchAllocationID[vSwitch.VSwitchId],
					})
				} else {
					zoneMappings = append(zoneMappings, alb.ZoneMapping{
						VSwitchId: vSwitch.VSwitchId,
						ZoneId:    vSwitch.ZoneId,
					})
				}
			}
		} else {
			vSwitches, err := t.vSwitchResolver.ResolveViaDiscovery(ctx)
			if err != nil {
				return alb.ALBLoadBalancerSpec{}, err
			}
			for _, vSwitch := range vSwitches {
				zoneMappings = append(zoneMappings, alb.ZoneMapping{
					VSwitchId: vSwitch.VSwitchId,
					ZoneId:    vSwitch.ZoneId,
				})
			}
		}
	}
	lbModel.ZoneMapping = zoneMappings

	lbModel.AddressAllocatedMode = albConfig.Spec.LoadBalancer.AddressAllocatedMode
	if lbModel.AddressAllocatedMode == "" {
		lbModel.AddressAllocatedMode = util.LoadBalancerAddressAllocatedModeDynamic
	}
	lbModel.AddressType = albConfig.Spec.LoadBalancer.AddressType
	if lbModel.AddressType == "" {
		lbModel.AddressType = util.LoadBalancerAddressTypeInternet
	}
	lbModel.Ipv6AddressType = albConfig.Spec.LoadBalancer.Ipv6AddressType
	if lbModel.Ipv6AddressType == "" {
		lbModel.Ipv6AddressType = util.LoadBalancerIpv6AddressTypeIntranet
	}
	lbModel.AddressIpVersion = albConfig.Spec.LoadBalancer.AddressIpVersion
	if lbModel.AddressIpVersion == "" {
		lbModel.AddressIpVersion = util.LoadBalancerAddressIpVersionIPv4
	}
	lbModel.DeletionProtectionConfig = alb.DeletionProtectionConfig{
		Enabled:     true,
		EnabledTime: "",
	}
	lbModel.ModificationProtectionConfig = alb.ModificationProtectionConfig{
		Reason: "",
		Status: util.LoadBalancerModificationProtectionStatusConsoleProtection,
	}
	payType := albConfig.Spec.LoadBalancer.BillingConfig.PayType
	if payType == "" {
		payType = util.LoadBalancerPayTypePostPay
	}
	lbModel.LoadBalancerBillingConfig = alb.LoadBalancerBillingConfig{
		InternetBandwidth:  albConfig.Spec.LoadBalancer.BillingConfig.InternetBandwidth,
		InternetChargeType: albConfig.Spec.LoadBalancer.BillingConfig.InternetChargeType,
		PayType:            payType,
		BandWidthPackageId: albConfig.Spec.LoadBalancer.BillingConfig.BandWidthPackageId,
	}
	lbModel.LoadBalancerEdition = albConfig.Spec.LoadBalancer.Edition
	if lbModel.LoadBalancerEdition == "" {
		lbModel.LoadBalancerEdition = util.LoadBalancerEditionStandard
	}

	// build customer tags
	if len(albConfig.Spec.LoadBalancer.Tags) != 0 {
		tags := make([]alb.ALBTag, 0)
		for _, tag := range albConfig.Spec.LoadBalancer.Tags {
			tags = append(tags, alb.ALBTag{
				Key:   tag.Key,
				Value: tag.Value,
			})
		}
		lbModel.Tags = tags
	}

	if albConfig.Spec.LoadBalancer.ResourceGroupId != "" {
		lbModel.ResourceGroupId = albConfig.Spec.LoadBalancer.ResourceGroupId
	}

	return lbModel, nil
}

var invalidLoadBalancerNamePattern = regexp.MustCompile("[[:^alnum:]]")

func (t *defaultModelBuildTask) buildAlbLoadBalancerName() (string, error) {
	uuidHash := sha256.New()
	_, _ = uuidHash.Write([]byte(t.clusterID))
	_, _ = uuidHash.Write([]byte(t.ingGroup.ID.String()))
	uuid := hex.EncodeToString(uuidHash.Sum(nil))

	sanitizedNamespace := invalidLoadBalancerNamePattern.ReplaceAllString(t.ingGroup.ID.Namespace, "")
	sanitizedName := invalidLoadBalancerNamePattern.ReplaceAllString(t.ingGroup.ID.Name, "")
	return fmt.Sprintf("k8s-%s-%s-%.10s", sanitizedNamespace, sanitizedName, uuid), nil
}

func (t *defaultModelBuildTask) buildAlbLoadBalancerStatus() alb.LoadBalancerStatus {
	return alb.LoadBalancerStatus{
		LoadBalancerID: t.albconfig.Status.LoadBalancer.Id,
		DNSName:        t.albconfig.Status.LoadBalancer.DNSName,
	}
}
