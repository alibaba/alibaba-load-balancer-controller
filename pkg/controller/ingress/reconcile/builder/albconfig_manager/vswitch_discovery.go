package albconfigmanager

import (
	"context"
	"time"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"

	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/go-logr/logr"
)

type VSwitchResolver interface {
	ResolveViaDiscovery(ctx context.Context) ([]vpc.VSwitch, error)

	ResolveViaIDSlice(ctx context.Context, vSwitchIDs []string) ([]vpc.VSwitch, error)
}

func NewDefaultVSwitchResolver(cloud prvd.Provider, vpcID string, logger logr.Logger) *defaultVSwitchesResolver {
	return &defaultVSwitchesResolver{
		cloud:  cloud,
		vpcID:  vpcID,
		logger: logger,
	}
}

var _ VSwitchResolver = &defaultVSwitchesResolver{}

type defaultVSwitchesResolver struct {
	cloud  prvd.Provider
	logger logr.Logger

	vpcID string
}

const (
	DescribeALBZones  = "DescribeALBZones"
	DescribeVSwitches = "DescribeVSwitches"
)

func (v *defaultVSwitchesResolver) discoveryALBZones(ctx context.Context) ([]albsdk.Zone, error) {
	traceID := ctx.Value(util.TraceID)

	req := albsdk.CreateDescribeZonesRequest()

	startTime := time.Now()
	v.logger.Info("describing alb zones",
		"traceID", traceID,
		"startTime", startTime,
		"action", DescribeALBZones)
	resp, err := v.cloud.DescribeALBZones(req)
	if err != nil {
		return nil, err
	}
	v.logger.Info("described alb zones",
		"traceID", traceID,
		"zones", resp.Zones,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		"requestID", resp.RequestId,
		"action", DescribeALBZones)

	return resp.Zones, nil
}

func (v *defaultVSwitchesResolver) ResolveViaDiscovery(ctx context.Context) ([]vpc.VSwitch, error) {
	traceID := ctx.Value(util.TraceID)

	albZones, err := v.discoveryALBZones(ctx)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	v.logger.Info("describing vSwitches",
		"traceID", traceID,
		"vpcID", v.vpcID,
		"startTime", startTime,
		"action", DescribeVSwitches)
	allVSwitches, err := v.cloud.DescribeVSwitches(ctx, v.vpcID)
	if err != nil {
		return nil, err
	}
	v.logger.Info("described vSwitches",
		"traceID", traceID,
		"vpcID", v.vpcID,
		"vSwitches", allVSwitches,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		"action", DescribeVSwitches)

	vSwitchesByZone := mapSDKVSwitchesByZone(allVSwitches)

	chosenVSwitches := make([]vpc.VSwitch, 0)

	for _, albZone := range albZones {
		if vSwitches, ok := vSwitchesByZone[albZone.ZoneId]; ok {
			for _, vSwitch := range vSwitches {
				// todo how to choose vSwitch
				chosenVSwitches = append(chosenVSwitches, vSwitch)
				break
			}
		}
	}

	return chosenVSwitches, nil
}

func mapSDKVSwitchesByZone(vSwitches []vpc.VSwitch) map[string][]vpc.VSwitch {
	vSwitchesByZone := make(map[string][]vpc.VSwitch)
	for _, vSwitch := range vSwitches {
		vSwitchesByZone[vSwitch.ZoneId] = append(vSwitchesByZone[vSwitch.ZoneId], vSwitch)
	}
	return vSwitchesByZone
}

func (v *defaultVSwitchesResolver) ResolveViaIDSlice(ctx context.Context, vSwitchIDs []string) ([]vpc.VSwitch, error) {
	traceID := ctx.Value(util.TraceID)

	startTime := time.Now()
	v.logger.Info("describing vSwitches",
		"traceID", traceID,
		"vpcID", v.vpcID,
		"vSwitchIDs", vSwitchIDs,
		"startTime", startTime,
		"action", DescribeVSwitches)
	allVSwitches, err := v.cloud.DescribeVSwitches(ctx, v.vpcID)
	if err != nil {
		return nil, err
	}
	v.logger.Info("described vSwitches",
		"traceID", traceID,
		"vpcID", v.vpcID,
		"vSwitches", allVSwitches,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		"action", DescribeVSwitches)

	chosenVSwitches := make([]vpc.VSwitch, 0)
	for _, vSwitchID := range vSwitchIDs {
		for _, vSwitch := range allVSwitches {
			if vSwitch.VSwitchId == vSwitchID {
				chosenVSwitches = append(chosenVSwitches, vSwitch)
				break
			}

		}
	}

	return chosenVSwitches, nil
}
