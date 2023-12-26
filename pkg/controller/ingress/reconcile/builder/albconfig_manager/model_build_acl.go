package albconfigmanager

import (
	"context"
	"fmt"

	v1 "k8s.io/alibaba-load-balancer-controller/pkg/apis/alibabacloud/v1"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
)

func (t *defaultModelBuildTask) buildAcl(ctx context.Context, ls *alb.Listener, lsSpec *v1.ListenerSpec, lb *alb.AlbLoadBalancer) error {

	aclTypeFlag := lsSpec.AclConfig.AclType
	var aclType string
	switch aclTypeFlag {
	case "White":
		aclType = util.AclTypeWhite
	case "Black":
		aclType = util.AclTypeBlack
	default:
		return nil
	}

	// cidr string to AclEntry
	entries := make([]alb.AclEntry, 0)
	for _, cidr := range lsSpec.AclConfig.AclEntries {
		entries = append(entries, alb.AclEntry{Entry: cidr})
	}

	if len(entries) > 0 && len(lsSpec.AclConfig.AclIds) > 0 {
		return fmt.Errorf("aclEntry and aclIds cannot use together")
	}

	aclName := lsSpec.AclConfig.AclName
	// auto aclName if aclEntry set
	if aclName == "" && len(entries) > 0 {
		aclName = "acl-" + lb.Spec.LoadBalancerName + "-" + lsSpec.Port.String()
	}
	aclSpec := &alb.AclSpec{
		ListenerID: ls.ListenerID(),
		AclName:    aclName,
		AclType:    aclType,
		AclEntries: entries,
		AclIds:     lsSpec.AclConfig.AclIds,
	}

	aclResID := fmt.Sprintf("%v", lsSpec.Port.String())
	alb.NewAcl(t.stack, aclResID, *aclSpec)
	return nil
}
