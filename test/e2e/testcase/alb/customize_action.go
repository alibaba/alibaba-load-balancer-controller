package alb

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/annotations"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/framework"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/testcase/alb/common"
	"k8s.io/klog/v2"
)

var (
	redirect      = "{\"type\":\"Redirect\",\"RedirectConfig\":{\"httpCode\":\"307\",\"port\":\"443\"}}"
	insertHeader  = "{\"type\":\"InsertHeader\",\"InsertHeaderConfig\":{\"key\":\"key\",\"value\":\"123\",\"valueType\":\"UserDefined\"}}"
	fixedResponse = "{\"type\":\"FixedResponse\",\"FixedResponseConfig\":{\"contentType\":\"text/plain\",\"httpCode\":\"503\",\"content\":\"503 error text\"}}"
	rewrite       = "{\"type\": \"Rewrite\",\"RewriteConfig\": {\"Host\": \"alb.ingress.top.com\",  \"Path\": \"/tea\",\"Query\": \"e2e-test\"} }"
	trafficlimit  = "{\"type\": \"TrafficLimit\",\"TrafficLimitConfig\": {\"QPS\": \"1000\",\"QPSPerIp\": \"100\"}}"
)

func RunCustomizeActionTestCases(f *framework.Framework) {
	//rule := common.Rule{}
	ingress := common.Ingress{}
	service := common.Service{}
	ginkgo.BeforeEach(func() {
		service.CreateDefaultService(f)
	})
	ginkgo.AfterEach(func() {
		ingress.DeleteIngress(f, ingress.DefaultIngress(f))
	})
	ginkgo.Describe("alb-ingress-controller: ingress", func() {
		ginkgo.Context("ingress create with customize action", func() {
			ginkgo.It("[alb][p0] ingress with existSgpForward action ", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 20 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardSingleTest := "[" + existSgpForward + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward": existSgpForwardSingleTest,
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward and InsertHeader action ", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardWithInsertHeader := "[" + existSgpForward + "," + insertHeader + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward": existSgpForwardWithInsertHeader,
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward and traffic-limit action ", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardSingleTest := "[" + existSgpForward + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					fmt.Sprintf(annotations.INGRESS_ALB_ACTIONS_ANNOTATIONS, serviceName): existSgpForwardSingleTest,
				}
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward":   existSgpForwardSingleTest,
					"alb.ingress.kubernetes.io/traffic-limit-qps": "50",
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward action and rewrite-target", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardSingleTest := "[" + existSgpForward + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					fmt.Sprintf(annotations.INGRESS_ALB_ACTIONS_ANNOTATIONS, serviceName): existSgpForwardSingleTest,
				}
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward": existSgpForwardSingleTest,
					"alb.ingress.kubernetes.io/rewrite-target":  "/path/${2}",
				}
				ing.Spec.Rules[0].HTTP.Paths[0].Path = "/something(/|$)(.*)"
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward action and FixedResponse/Redirect/InsertHeader action", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardSingleTest := "[" + existSgpForward + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward":       existSgpForwardSingleTest,
					"alb.ingress.kubernetes.io/actions.response-503":  "[{\"type\":\"FixedResponse\",\"FixedResponseConfig\":{\"contentType\":\"text/plain\",\"httpCode\":\"503\",\"content\":\"503 error text\"}}]",
					"alb.ingress.kubernetes.io/actions.redirect":      "[{\"type\":\"Redirect\",\"RedirectConfig\":{\"httpCode\":\"307\",\"port\":\"443\"}}]",
					"alb.ingress.kubernetes.io/actions.insert-header": "[{\"type\":\"InsertHeader\",\"InsertHeaderConfig\":{\"key\":\"key\",\"value\":\"value\",\"valueType\":\"UserDefined\"}},{\"type\":\"Redirect\",\"RedirectConfig\":{\"httpCode\":\"307\",\"port\":\"443\"}}]",
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "response-503"))
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path2", "redirect"))
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path3", "insert-header"))
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path4", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward action and all customize condition", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardSingleTest := "[" + existSgpForward + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					fmt.Sprintf(annotations.INGRESS_ALB_CONDITIONS_ANNOTATIONS, serviceName): allConditions,
					"alb.ingress.kubernetes.io/actions.forward":                              existSgpForwardSingleTest,
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward action and canary-by-header", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardSingleTest := "[" + existSgpForward + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward":        existSgpForwardSingleTest,
					"alb.ingress.kubernetes.io/canary":                 "true",
					"alb.ingress.kubernetes.io/canary-by-header":       "location",
					"alb.ingress.kubernetes.io/canary-by-header-value": "hz",
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward action and canary-by-weight", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardSingleTest := "[" + existSgpForward + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward": existSgpForwardSingleTest,
					"alb.ingress.kubernetes.io/canary":          "true",
					"alb.ingress.kubernetes.io/canary-weight":   "50",
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, false)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward action and rewrite-action", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardWithRewrite := "[" + existSgpForward + "," + rewrite + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward": existSgpForwardWithRewrite,
				}
				ing.Spec.Rules[0].HTTP.Paths[0].Path = "/something"
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward and Trafficlimit customize action ", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardWithTrafficlimit := "[" + existSgpForward + "," + trafficlimit + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward": existSgpForwardWithTrafficlimit,
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
				ingress.DeleteIngress(f, defaultIngress(f))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward and traffic-limit action in both annotation and customize action", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpTrafficLimitConflict := "[" + existSgpForward + "," + trafficlimit + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					fmt.Sprintf(annotations.INGRESS_ALB_ACTIONS_ANNOTATIONS, serviceName): existSgpTrafficLimitConflict,
				}
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward":   existSgpTrafficLimitConflict,
					"alb.ingress.kubernetes.io/traffic-limit-qps": "5000",
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, false)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
			})
			ginkgo.It("[alb][p0] ingress with existSgpForward and InsertHeader action and Trafficlimit customize action", func() {
				service.WaitCreateDefaultServiceWithSvcName(f, "exist-sgp-svc")
				sgpIng := defaultIngressWithSvcName(f, "exist-sgp-svc")
				ingress.WaitCreateIngress(f, sgpIng, true)
				tags, _ := buildServerGroupTags(sgpIng)
				klog.Infof("%s %s %s %s", tags[util.ServiceNamespaceTagKey], tags[util.IngressNameTagKey], tags[util.ServiceNameTagKey], tags[util.ServicePortTagKey])
				existServerGroupId, _ := findTrafficMirrorServerGroupId(f, tags)
				existSgpForward := "{\"type\": \"ForwardGroup\",\"ForwardConfig\": {\"ServerGroups\" : [{\"ServerGroupID\": \"" + existServerGroupId + "\", \"Weight\": 100 },{\"ServiceName\": \"tea-svc\",\"Weight\": 80,\"ServicePort\": 80 }] } }"
				existSgpForwardTrafficlimitWithInsertHeader := "[" + existSgpForward + "," + insertHeader + "," + trafficlimit + "]"
				ing := ingress.DefaultIngress(f)
				ing.Annotations = map[string]string{
					"alb.ingress.kubernetes.io/actions.forward": existSgpForwardTrafficlimitWithInsertHeader,
				}
				ing.Spec.Rules = append(ing.Spec.Rules, defaultRule("/path1", "forward"))
				ingress.WaitCreateIngress(f, ing, true)
				ingress.DeleteIngress(f, defaultIngressWithSvcName(f, "exist-sgp-svc"))
				ingress.DeleteIngress(f, defaultIngress(f))
			})
		})
	})
}
