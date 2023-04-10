package context

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller/service/reconcile/annotation"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

type RequestContext struct {
	Ctx      context.Context
	Service  *v1.Service
	Anno     *annotation.AnnotationRequest
	Log      logr.Logger
	Recorder record.EventRecorder
}
