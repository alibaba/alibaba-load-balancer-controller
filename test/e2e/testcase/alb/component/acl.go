package component

import (
	"context"

	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/alb"
)

type AclTest struct {
	ListenerId string
	AclId      string
}

func (a *AclTest) DeleteAcl(alb *alb.ALBProvider) error {
	ctx := context.TODO()
	return alb.DeleteAcl(ctx, a.ListenerId, a.AclId)
}
