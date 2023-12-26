package applier

import (
	"context"
	"fmt"
	"sync"

	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"github.com/go-logr/logr"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/tracking"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb/core"
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
	"k8s.io/apimachinery/pkg/util/sets"
)

func NewAclApplier(albProvider prvd.Provider, trackingProvider tracking.TrackingProvider, stack core.Manager, logger logr.Logger, errRes core.ErrResult) *aclApplier {
	return &aclApplier{
		albProvider:      albProvider,
		trackingProvider: trackingProvider,
		stack:            stack,
		logger:           logger,
		errRes:           errRes,
	}
}

type aclApplier struct {
	albProvider      prvd.Provider
	trackingProvider tracking.TrackingProvider
	stack            core.Manager
	logger           logr.Logger
	errRes           core.ErrResult
}

type MatchResAndSDKAcl struct {
	MatchedResAndSDKAcls []alb.ResAndSDKAclPair
	UnmatchedResAcls     []*alb.Acl
	UnmatchedSDKAcls     []albsdk.Acl
}

func (s *aclApplier) Apply(ctx context.Context) error {
	var resAcls []*alb.Acl
	s.stack.ListResources(&resAcls)

	resAclsByLsID, err := mapResAclByListenerID(ctx, resAcls)
	if err != nil {
		return err
	}

	var resLSs []*alb.Listener
	s.stack.ListResources(&resLSs)

	resLSsByLsID, err := mapResListenerByListenerID(ctx, resLSs)
	if err != nil {
		return err
	}

	var (
		errSynthesize error
		wgSynthesize  sync.WaitGroup
		chSynthesize  = make(chan struct{}, util.ListenerConcurrentNum)
	)

	for lsID, resLS := range resLSsByLsID {
		err := s.errRes.CheckErrMsgsByListenerPort(resLS.Spec.ListenerPort)
		if err != nil {
			s.logger.V(util.SynLogLevel).Error(err, "CheckErrMsgsByListenerPort succ")
			continue
		}
		chSynthesize <- struct{}{}
		wgSynthesize.Add(1)

		go func(listenerID string) {
			util.RandomSleepFunc(util.ConcurrentMaxSleepMillisecondTime)

			defer func() {
				wgSynthesize.Done()
				<-chSynthesize
			}()

			acls := resAclsByLsID[listenerID]
			if errOnce := s.synthesizeAclsOnListener(ctx, resLSsByLsID[listenerID], acls); errSynthesize == nil && errOnce != nil {
				s.logger.Error(errOnce, "synthesize acl failed", "listener", listenerID)
				errSynthesize = errOnce
				return
			}
		}(lsID)
	}
	wgSynthesize.Wait()
	if errSynthesize != nil {
		return errSynthesize
	}

	return nil
}
func (s *aclApplier) PostApply(ctx context.Context) error {
	return nil
}

func (s *aclApplier) synthesizeAclsOnListener(ctx context.Context, listener *alb.Listener, resAcl *alb.Acl) error {
	if listener == nil {
		return fmt.Errorf("empty listenerwhen synthesize acls error")
	}
	traceID := ctx.Value(util.TraceID)
	lsId, err := listener.ListenerID().Resolve(ctx)
	if err != nil {
		return err
	}
	aclIds, sdkAclType, err := s.findListenerAclConfig(ctx, lsId)
	if err != nil {
		return err
	}
	if resAcl == nil {
		return nil
	}
	resAclType := resAcl.Spec.AclType
	if resAcl.Spec.AclName == "" {
		// take aclIds with other unmatchedSDKAcls
		_, unmatchedResAclIds, unmatchedSdkAclIds := matchResAndSDKAclIds(resAcl.Spec.AclIds, aclIds)
		if resAclType != sdkAclType {
			unmatchedResAclIds = resAcl.Spec.AclIds
			unmatchedSdkAclIds = aclIds
		}

		if len(unmatchedResAclIds) > 0 {
			s.logger.V(util.SynLogLevel).Info("synthesize aclIds",
				"unmatchedResAclIds", unmatchedResAclIds,
				"traceID", traceID)
			if err := s.albProvider.AssociateAclWithListener(ctx, traceID, resAcl, unmatchedResAclIds); err != nil {
				return err
			}
		}
		if len(unmatchedSdkAclIds) > 0 {
			s.logger.V(util.SynLogLevel).Info("synthesize aclIds",
				"unmatchedSdkAclIds", unmatchedSdkAclIds,
				"traceID", traceID)
			if err := s.albProvider.DisassociateAclWithListener(traceID, lsId, unmatchedSdkAclIds); err != nil {
				return err
			}
		}

	} else {
		sdkAcls, err := s.findSDKAclsOnLS(ctx, listener, aclIds)
		if err != nil {
			return err
		}
		matchedResAndSDKAcls, unmatchedResAcls, unmatchedSDKAcls := matchResAndSDKAcls([]*alb.Acl{resAcl}, sdkAcls)
		// acl type change need re-related operation
		if resAclType != sdkAclType {
			unmatchedSDKAcls = sdkAcls
			unmatchedResAcls = []*alb.Acl{resAcl}
			matchedResAndSDKAcls = make([]alb.ResAndSDKAclPair, 0)
		}
		if len(matchedResAndSDKAcls) != 0 {
			s.logger.V(util.SynLogLevel).Info("synthesize acls",
				"matchedResAndSDKAcls", matchedResAndSDKAcls,
				"traceID", traceID)
		}
		if len(unmatchedResAcls) != 0 {
			s.logger.V(util.SynLogLevel).Info("synthesize acls",
				"unmatchedResAcls", unmatchedResAcls,
				"traceID", traceID)
		}
		if len(unmatchedSDKAcls) != 0 {
			s.logger.V(util.SynLogLevel).Info("synthesize acls",
				"unmatchedSDKAcls", unmatchedSDKAcls,
				"traceID", traceID)
		}
		if err := s.createAndUpdateMatchedAcl(ctx, resAclType, lsId, listener, MatchResAndSDKAcl{
			MatchedResAndSDKAcls: matchedResAndSDKAcls,
			UnmatchedResAcls:     unmatchedResAcls,
			UnmatchedSDKAcls:     unmatchedSDKAcls,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (a *aclApplier) createAndUpdateMatchedAcl(ctx context.Context, sdkAclType, lsId string, listener *alb.Listener, match MatchResAndSDKAcl) error {
	for _, resAcl := range match.UnmatchedResAcls {
		aclStatus, err := a.albProvider.CreateAcl(ctx, resAcl)
		if err != nil {
			return err
		}
		resAcl.SetStatus(aclStatus)
	}
	for _, resAndSDKAcl := range match.MatchedResAndSDKAcls {
		aclStatus, err := a.albProvider.UpdateAcl(ctx, lsId, resAndSDKAcl)
		if err != nil {
			return err
		}
		resAndSDKAcl.ResAcl.SetStatus(aclStatus)
	}
	return nil
}

func (s *aclApplier) findListenerAclConfig(ctx context.Context, lsId string) ([]string, string, error) {
	lsAttrResponse, err := s.albProvider.GetALBListenerAttribute(ctx, lsId)
	if err != nil {
		return nil, "", err
	}
	if len(lsAttrResponse.AclConfig.AclRelations) == 0 {
		return nil, "", nil
	}
	aclConfig := lsAttrResponse.AclConfig
	aclIds := []string{}
	for _, acl := range aclConfig.AclRelations {
		aclIds = append(aclIds, acl.AclId)
	}
	return aclIds, aclConfig.AclType, nil
}

func (s *aclApplier) findSDKAclsOnLS(ctx context.Context, listener *alb.Listener, aclIds []string) ([]albsdk.Acl, error) {
	acls, err := s.albProvider.ListAcl(ctx, listener, aclIds)
	if err != nil {
		return nil, err
	}
	return acls, nil
}

func matchResAndSDKAclIds(resAclIds, sdkAclIds []string) ([]string, []string, []string) {

	sdkAclIdsSet := make(sets.String, len(sdkAclIds))
	for _, sdkAclId := range sdkAclIds {
		sdkAclIdsSet[sdkAclId] = sets.Empty{}
	}
	resAclIdsSet := make(sets.String, len(resAclIds))
	for _, resAclId := range resAclIds {
		resAclIdsSet[resAclId] = sets.Empty{}
	}
	matchedResAndSdkAclIds := resAclIdsSet.Intersection(sdkAclIdsSet).List()
	unmatchedResAclIds := resAclIdsSet.Difference(sdkAclIdsSet).List()
	unmatchedSdkAclIds := sdkAclIdsSet.Difference(resAclIdsSet).List()
	return matchedResAndSdkAclIds, unmatchedResAclIds, unmatchedSdkAclIds
}

func matchResAndSDKAcls(resAcls []*alb.Acl, sdkAcls []albsdk.Acl) ([]alb.ResAndSDKAclPair, []*alb.Acl, []albsdk.Acl) {
	var matchedResAndSDKAcls []alb.ResAndSDKAclPair
	var unmatchedResAcls []*alb.Acl
	var unmatchedSDKAcls []albsdk.Acl

	resAclsMap := mapResAclByName(resAcls)
	sdkAclsMap := mapSdkAclByName(sdkAcls)

	resAclsSet := sets.StringKeySet(resAclsMap)
	sdkAclsSet := sets.StringKeySet(sdkAclsMap)

	for _, aclName := range resAclsSet.Intersection(sdkAclsSet).List() {
		resAcl := resAclsMap[aclName]
		sdkAcl := sdkAclsMap[aclName]
		matchedResAndSDKAcls = append(matchedResAndSDKAcls, alb.ResAndSDKAclPair{
			ResAcl: resAcl,
			SdkAcl: &sdkAcl,
		})
	}
	for _, aclName := range resAclsSet.Difference(sdkAclsSet).List() {
		unmatchedResAcls = append(unmatchedResAcls, resAclsMap[aclName])
	}
	for _, aclName := range sdkAclsSet.Difference(resAclsSet).List() {
		unmatchedSDKAcls = append(unmatchedSDKAcls, sdkAclsMap[aclName])
	}
	return matchedResAndSDKAcls, unmatchedResAcls, unmatchedSDKAcls
}

func mapResAclByName(resAcls []*alb.Acl) map[string]*alb.Acl {
	resAclByName := make(map[string]*alb.Acl, 0)
	for _, resAcl := range resAcls {
		resAclByName[resAcl.Spec.AclName] = resAcl
	}
	return resAclByName
}
func mapSdkAclByName(sdkAcls []albsdk.Acl) map[string]albsdk.Acl {
	sdkAclByName := make(map[string]albsdk.Acl, 0)
	for _, sdkAcl := range sdkAcls {
		sdkAclByName[sdkAcl.AclName] = sdkAcl
	}
	return sdkAclByName
}

func mapResAclByListenerID(ctx context.Context, resAcls []*alb.Acl) (map[string]*alb.Acl, error) {
	resAclByLsID := make(map[string]*alb.Acl, 0)
	for _, acl := range resAcls {
		lsID, err := acl.Spec.ListenerID.Resolve(ctx)
		if err != nil {
			return nil, err
		}
		resAclByLsID[lsID] = acl
	}
	return resAclByLsID, nil
}
