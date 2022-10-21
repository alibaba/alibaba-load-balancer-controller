package helper

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	ctrlCfg "k8s.io/alibaba-load-balancer-controller/pkg/config"
	"k8s.io/alibaba-load-balancer-controller/pkg/model"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AnnotationLoadBalancerPrefix = "loadbalancer-"
	RemoveUnscheduled            = AnnotationLoadBalancerPrefix + "remove-unscheduled-backend"
	BackendLabel                 = AnnotationLoadBalancerPrefix + "backend-label"
	BackendType                  = "service.beta.kubernetes.io/backend-type"
	LabelNodeTypeVK              = "virtual-kubelet"
	LabelNodeRoleMaster          = "node-role.kubernetes.io/master"
	// AnnotationLegacyPrefix legacy prefix of service annotation
	AnnotationLegacyPrefix = "service.beta.kubernetes.io/alicloud"
	// AnnotationPrefix prefix of service annotation
	AnnotationPrefix = "service.beta.kubernetes.io/alibaba-cloud"
)

type TrafficPolicy string

const (
	// LocalTrafficPolicy externalTrafficPolicy=Local
	LocalTrafficPolicy = TrafficPolicy("Local")
	// ClusterTrafficPolicy externalTrafficPolicy=Cluster
	ClusterTrafficPolicy = TrafficPolicy("Cluster")
	// ENITrafficPolicy is forwarded to pod directly
	ENITrafficPolicy = TrafficPolicy("ENI")
)

type RequestContext struct {
	Ctx      context.Context
	Service  *v1.Service
	Anno     *AnnotationRequest
	Log      logr.Logger
	Recorder record.EventRecorder
}

type AnnotationRequest struct{ Service *v1.Service }

func (n *AnnotationRequest) Get(k string) string {
	if n.Service == nil {
		return ""
	}

	if n.Service.Annotations == nil {
		return ""
	}

	key := composite(AnnotationPrefix, k)
	v, ok := n.Service.Annotations[key]
	if ok {
		return v
	}

	lkey := composite(AnnotationLegacyPrefix, k)
	v, ok = n.Service.Annotations[lkey]
	if ok {
		return v
	}

	return ""
}

func composite(p, k string) string {
	return fmt.Sprintf("%s-%s", p, k)
}

func GetNodes(reqCtx *RequestContext, client client.Client) ([]v1.Node, error) {
	nodeList := v1.NodeList{}
	err := client.List(reqCtx.Ctx, &nodeList)
	if err != nil {
		return nil, fmt.Errorf("get nodes error: %s", err.Error())
	}

	// 1. filter by label
	items := nodeList.Items
	if reqCtx.Anno.Get(BackendLabel) != "" {
		items, err = filterOutByLabel(nodeList.Items, reqCtx.Anno.Get(BackendLabel))
		if err != nil {
			return nil, fmt.Errorf("filter nodes by label error: %s", err.Error())
		}
	}

	var nodes []v1.Node
	for _, n := range items {
		if needExcludeFromLB(reqCtx, &n) {
			continue
		}
		nodes = append(nodes, n)
	}

	return nodes, nil
}
func needExcludeFromLB(reqCtx *RequestContext, node *v1.Node) bool {
	// need to keep the node who has exclude label in order to be compatible with vk node
	// It's safe because these nodes will be filtered in build backends func

	if isMasterNode(node) {
		klog.V(5).Infof("[%s] node %s is master node, skip adding it to lb", util.Key(reqCtx.Service), node.Name)
		return true
	}

	// filter unscheduled node
	if node.Spec.Unschedulable && reqCtx.Anno.Get(RemoveUnscheduled) != "" {
		if reqCtx.Anno.Get(RemoveUnscheduled) == string(model.OnFlag) {
			reqCtx.Log.Info("node is unschedulable, skip add to lb", "node", node.Name)
			return true
		}
	}

	// ignore vk node condition check.
	// Even if the vk node is NotReady, it still can be added to lb. Because the eci pod that actually joins the lb, not a vk node
	if label, ok := node.Labels["type"]; ok && label == LabelNodeTypeVK {
		return false
	}

	// If we have no info, don't accept
	if len(node.Status.Conditions) == 0 {
		reqCtx.Log.Info("node condition is nil, skip add to lb", "node", node.Name)
		return true
	}

	for _, cond := range node.Status.Conditions {
		// We consider the node for load balancing only when its NodeReady
		// condition status is ConditionTrue
		if cond.Type == v1.NodeReady &&
			cond.Status != v1.ConditionTrue {
			reqCtx.Log.Info(fmt.Sprintf("node not ready with %v condition, status %v", cond.Type, cond.Status),
				"node", node.Name)
			return true
		}
	}

	return false
}

func isMasterNode(node *v1.Node) bool {
	if _, isMaster := node.Labels[LabelNodeRoleMaster]; isMaster {
		return true
	}
	return false
}

func filterOutByLabel(nodes []v1.Node, labels string) ([]v1.Node, error) {
	if labels == "" {
		return nodes, nil
	}
	var result []v1.Node
	lbl := strings.Split(labels, ",")
	var records []string
	for _, node := range nodes {
		found := true
		for _, v := range lbl {
			l := strings.Split(v, "=")
			if len(l) < 2 {
				return []v1.Node{}, fmt.Errorf("parse backend label: %s, [k1=v1,k2=v2]", v)
			}
			if nv, exist := node.Labels[l[0]]; !exist || nv != l[1] {
				found = false
				break
			}
		}
		if found {
			result = append(result, node)
			records = append(records, node.Name)
		}
	}
	klog.V(4).Infof("accept nodes backend labels[%s], %v", labels, records)
	return result, nil
}

func isLocalModeService(svc *v1.Service) bool {
	return svc.Spec.ExternalTrafficPolicy == v1.ServiceExternalTrafficPolicyTypeLocal
}

func isClusterIPService(svc *v1.Service) bool {
	return svc.Spec.Type == v1.ServiceTypeClusterIP
}

func isENIBackendType(svc *v1.Service) bool {
	if svc.Annotations[BackendType] != "" {
		return svc.Annotations[BackendType] == model.ENIBackendType
	}

	if os.Getenv("SERVICE_FORCE_BACKEND_ENI") != "" {
		return os.Getenv("SERVICE_FORCE_BACKEND_ENI") == "true"
	}

	return ctrlCfg.CloudCFG.Global.ServiceBackendType == model.ENIBackendType
}

func GetServiceTrafficPolicy(svc *v1.Service) (TrafficPolicy, error) {
	if isENIBackendType(svc) {
		return ENITrafficPolicy, nil
	}
	if isClusterIPService(svc) {
		return "", fmt.Errorf("cluster service type just support eni mode for alb ingress")
	}
	if isLocalModeService(svc) {
		return LocalTrafficPolicy, nil
	}
	return ClusterTrafficPolicy, nil
}

// providerID
// 1) the id of the instance in the alicloud API. Use '.' to separate providerID which looks like 'cn-hangzhou.i-v98dklsmnxkkgiiil7'. The format of "REGION.NODEID"
// 2) the id for an instance in the kubernetes API, which has 'alicloud://' prefix. e.g. alicloud://cn-hangzhou.i-v98dklsmnxkkgiiil7
func NodeFromProviderID(providerID string) (string, string, error) {
	if strings.HasPrefix(providerID, "alicloud://") {
		k8sName := strings.Split(providerID, "://")
		if len(k8sName) < 2 {
			return "", "", fmt.Errorf("alicloud: unable to split instanceid and region from providerID, error unexpected providerID=%s", providerID)
		} else {
			providerID = k8sName[1]
		}
	}

	name := strings.Split(providerID, ".")
	if len(name) < 2 {
		return "", "", fmt.Errorf("alicloud: unable to split instanceid and region from providerID, error unexpected providerID=%s", providerID)
	}
	return name[0], name[1], nil
}
