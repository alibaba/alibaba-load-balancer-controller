package albconfigmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/annotations"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb/core"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"

	"github.com/pkg/errors"
)

const (
	ListenerRuleNamePrefix = "rule"
	HTTPRedirectCode       = "308"
	CookieAlways           = "always"
	HTTPS443               = "443"
	KnativeIngress         = "knative.aliyun.com/ingress"
)

var (
	actionTypeMap = map[string]string{
		util.RuleActionTypeFixedResponse: "FixedResponseConfig",
		util.RuleActionTypeRedirect:      "RedirectConfig",
		util.RuleActionTypeInsertHeader:  "InsertHeaderConfig",
	}
)

func (t *defaultModelBuildTask) buildListenerRules(ctx context.Context, lsID core.StringToken, port int32, ingList []networking.Ingress) error {
	if len(ingList) > 0 {
		ing := ingList[0]
		if _, ok := ing.Labels[KnativeIngress]; ok {
			return t.buildListenerRulesCommon(ctx, lsID, port, ingList)
		}
	}
	var rules []alb.ListenerRule
	carryWeight := make(map[string][]alb.ServerGroupTuple, 0)
	nonCanaryPath := make(map[string]bool, 0)
	for _, ing := range ingList {
		if v := annotations.GetStringAnnotationMutil(annotations.NginxCanary, annotations.AlbCanary, &ing); v == "true" {
			weight := getIfOnlyWeight(&ing)
			if weight == 0 {
				continue
			}
			for _, rule := range ing.Spec.Rules {
				if rule.HTTP == nil {
					continue
				}
				for _, path := range rule.HTTP.Paths {
					if _, ok := carryWeight[rule.Host+"-"+path.Path]; !ok {
						carryWeight[rule.Host+"-"+path.Path] = make([]alb.ServerGroupTuple, 0)
					}
					carryWeight[rule.Host+"-"+path.Path] = append(carryWeight[rule.Host+"-"+path.Path], alb.ServerGroupTuple{
						ServiceName: path.Backend.Service.Name,
						ServicePort: int(path.Backend.Service.Port.Number),
						Weight:      weight,
					})
				}
			}
		} else {
			for _, rule := range ing.Spec.Rules {
				if rule.HTTP == nil {
					continue
				}
				for _, path := range rule.HTTP.Paths {
					nonCanaryPath[rule.Host+"-"+path.Path] = true
				}
			}
		}
	}
	for _, ing := range ingList {
		if v := annotations.GetStringAnnotationMutil(annotations.NginxCanary, annotations.AlbCanary, &ing); v == "true" {
			we := annotations.GetStringAnnotationMutil(annotations.NginxCanaryWeight, annotations.AlbCanaryWeight, &ing)
			if we != "" {
				continue
			}
		}
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				if _, ok := nonCanaryPath[rule.Host+"-"+path.Path]; !ok {
					continue
				}
				actions, err := t.buildRuleActions(ctx, &ing, &path, carryWeight[rule.Host+"-"+path.Path], port == 443)
				if err != nil {
					return errors.Wrapf(err, "buildListenerRules-Actions(ingress: %v)", util.NamespacedName(&ing))
				}
				conditions, err := t.buildRuleConditions(ctx, rule, path, ing)
				if err != nil {
					return errors.Wrapf(err, "buildListenerRules-Actions(ingress: %v)", util.NamespacedName(&ing))
				}
				rules = append(rules, alb.ListenerRule{
					Spec: alb.ListenerRuleSpec{
						ListenerID: lsID,
						ALBListenerRuleSpec: alb.ALBListenerRuleSpec{
							RuleActions:    actions,
							RuleConditions: conditions,
						},
					},
				})
			}
		}
	}

	priority := 1
	for _, rule := range rules {
		ruleResID := fmt.Sprintf("%v:%v", port, priority)
		klog.Infof("ruleResID: %s", ruleResID)
		lrs := alb.ListenerRuleSpec{
			ListenerID: lsID,
			ALBListenerRuleSpec: alb.ALBListenerRuleSpec{
				Priority:       priority,
				RuleConditions: rule.Spec.RuleConditions,
				RuleActions:    rule.Spec.RuleActions,
				RuleName:       fmt.Sprintf("%v-%v-%v", ListenerRuleNamePrefix, port, priority),
			},
		}
		_ = alb.NewListenerRule(t.stack, ruleResID, lrs)
		priority += 1
	}

	return nil
}

func (t *defaultModelBuildTask) buildListenerRulesCommon(ctx context.Context, lsID core.StringToken, port int32, ingList []networking.Ingress) error {
	var rules []alb.ListenerRule
	for _, ing := range ingList {
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				var action alb.Action
				actionStr := ing.Annotations[fmt.Sprintf(annotations.INGRESS_ALB_ACTIONS_ANNOTATIONS, path.Backend.Service.Name)]
				if actionStr != "" {
					err := json.Unmarshal([]byte(actionStr), &action)
					if err != nil {
						klog.Errorf("buildListenerRulesCommon: %s Unmarshal: %s", actionStr, err.Error())
						continue
					}
				}
				klog.Infof("INGRESS_ALB_ACTIONS_ANNOTATIONS: %s", actionStr)
				conditions, err := t.buildRuleConditionsCommon(ctx, rule, path, ing)
				if err != nil {
					klog.Errorf("buildListenerRulesCommon error: %s", err.Error())
					continue
				}
				action2, err := t.buildAction(ctx, ing, action)
				if err != nil {
					klog.Errorf("buildListenerRulesCommon error: %s", err.Error())
					continue
				}

				lrs := alb.ListenerRuleSpec{
					ListenerID: lsID,
				}
				lrs.RuleActions = []alb.Action{action2}
				lrs.RuleConditions = conditions
				rules = append(rules, alb.ListenerRule{
					Spec: lrs,
				})
			}
		}
	}

	priority := 1
	for _, rule := range rules {
		ruleResID := fmt.Sprintf("%v:%v", port, priority)
		klog.Infof("ruleResID: %s", ruleResID)
		lrs := alb.ListenerRuleSpec{
			ListenerID: lsID,
		}
		lrs.Priority = priority
		lrs.RuleConditions = rule.Spec.RuleConditions
		lrs.RuleActions = rule.Spec.RuleActions
		lrs.RuleName = fmt.Sprintf("%v-%v-%v", ListenerRuleNamePrefix, port, priority)
		_ = alb.NewListenerRule(t.stack, ruleResID, lrs)
		priority += 1
	}

	return nil
}
func buildActionViaServiceAndServicePort(_ context.Context, svcName string, svcPort int, weight int) alb.Action {
	action := alb.Action{
		Type: util.RuleActionTypeForward,
		ForwardConfig: &alb.ForwardActionConfig{
			ServerGroups: []alb.ServerGroupTuple{
				{
					ServiceName: svcName,
					ServicePort: svcPort,
					Weight:      weight,
				},
			},
		},
	}
	return action
}

func buildActionViaHostAndPath(_ context.Context, path string) alb.Action {
	action := alb.Action{
		Type: util.RuleActionTypeRedirect,
		RedirectConfig: &alb.RedirectConfig{
			Host:     "${host}",
			Path:     "${path}",
			Protocol: util.ListenerProtocolHTTPS,
			Port:     HTTPS443,
			HttpCode: HTTPRedirectCode,
			Query:    "${query}",
		},
	}
	return action
}

func (t *defaultModelBuildTask) buildPathPatternsForImplementationSpecificPathType(path string) ([]string, error) {
	return []string{path}, nil
}

func (t *defaultModelBuildTask) buildPathPatternsForExactPathType(path string) ([]string, error) {
	if strings.ContainsAny(path, "*?") {
		return nil, errors.Errorf("exact path shouldn't contain wildcards: %v", path)
	}
	return []string{path}, nil
}

func (t *defaultModelBuildTask) buildPathPatternsForPrefixPathType(path string, ing networking.Ingress) ([]string, error) {
	if path == "/" {
		return []string{"/*"}, nil
	}
	useRegex := false
	if v, err := annotations.GetStringAnnotation(annotations.AlbRewriteTarget, &ing); err == nil && v != path {
		useRegex = true
	}
	var paths []string
	if useRegex {
		paths = []string{"~*" + path}
	} else {
		if strings.ContainsAny(path, "*?") {
			return nil, errors.Errorf("prefix path shouldn't contain wildcards: %v", path)
		}
		normalizedPath := strings.TrimSuffix(path, "/")
		paths = []string{normalizedPath, normalizedPath + "/*"}
	}
	return paths, nil
}

func (t *defaultModelBuildTask) buildPathPatterns(path string, pathType *networking.PathType, ing networking.Ingress) ([]string, error) {
	normalizedPathType := networking.PathTypeImplementationSpecific
	if pathType != nil {
		normalizedPathType = *pathType
	}
	switch normalizedPathType {
	case networking.PathTypeImplementationSpecific:
		return t.buildPathPatternsForImplementationSpecificPathType(path)
	case networking.PathTypeExact:
		return t.buildPathPatternsForExactPathType(path)
	case networking.PathTypePrefix:
		return t.buildPathPatternsForPrefixPathType(path, ing)
	default:
		return nil, errors.Errorf("unsupported pathType: %v", normalizedPathType)
	}
}

func (t *defaultModelBuildTask) buildHostHeaderCondition(_ context.Context, hosts []string) alb.Condition {
	return alb.Condition{
		Type: util.RuleConditionFieldHost,
		HostConfig: alb.HostConfig{
			Values: hosts,
		},
	}
}

func (t *defaultModelBuildTask) buildHeaderCondition(_ context.Context, key string, values []string) alb.Condition {
	return alb.Condition{
		Type: util.RuleConditionFieldHeader,
		HeaderConfig: alb.HeaderConfig{
			Key:    key,
			Values: values,
		},
	}
}
func (t *defaultModelBuildTask) buildCookieCondition(_ context.Context, key string, value string) alb.Condition {
	return alb.Condition{
		Type: util.RuleConditionFieldCookie,
		CookieConfig: alb.CookieConfig{
			Values: []alb.Value{{
				Key:   key,
				Value: value,
			},
			},
		},
	}
}

func (t *defaultModelBuildTask) buildPathPatternCondition(_ context.Context, paths []string) alb.Condition {
	return alb.Condition{
		Type: util.RuleConditionFieldPath,
		PathConfig: alb.PathConfig{
			Values: paths,
		},
	}
}

func (t *defaultModelBuildTask) buildRuleConditions(ctx context.Context, rule networking.IngressRule,
	path networking.HTTPIngressPath, ing networking.Ingress) ([]alb.Condition, error) {
	var hosts []string
	if rule.Host != "" {
		hosts = append(hosts, rule.Host)
	}
	var paths []string
	if path.Path != "" {
		pathPatterns, err := t.buildPathPatterns(path.Path, path.PathType, ing)
		if err != nil {
			return nil, err
		}
		paths = append(paths, pathPatterns...)
	}

	var conditions []alb.Condition
	if len(hosts) != 0 {
		conditions = append(conditions, t.buildHostHeaderCondition(ctx, hosts))
	}
	if len(paths) != 0 {
		conditions = append(conditions, t.buildPathPatternCondition(ctx, paths))
	}
	if v := annotations.GetStringAnnotationMutil(annotations.NginxCanary, annotations.AlbCanary, &ing); v == "true" {
		header := annotations.GetStringAnnotationMutil(annotations.NginxCanaryByHeader, annotations.AlbCanaryByHeader, &ing)
		if header != "" {
			value := annotations.GetStringAnnotationMutil(annotations.NginxCanaryByHeaderValue, annotations.AlbCanaryByHeaderValue, &ing)
			conditions = append(conditions, t.buildHeaderCondition(ctx, header, []string{value}))
		}
		cookie := annotations.GetStringAnnotationMutil(annotations.NginxCanaryByCookie, annotations.AlbCanaryByCookie, &ing)
		if cookie != "" {
			conditions = append(conditions, t.buildCookieCondition(ctx, cookie, CookieAlways))
		}

	}

	if len(conditions) == 0 {
		conditions = append(conditions, t.buildPathPatternCondition(ctx, []string{"/*"}))
	}

	return conditions, nil
}

func (t *defaultModelBuildTask) buildRuleConditionsCommon(ctx context.Context, rule networking.IngressRule,
	path networking.HTTPIngressPath, ing networking.Ingress) ([]alb.Condition, error) {
	var hosts []string
	if rule.Host != "" {
		hosts = append(hosts, rule.Host)
	}
	var paths []string
	if path.Path != "" {
		pathPatterns, err := t.buildPathPatterns(path.Path, path.PathType, ing)
		if err != nil {
			return nil, err
		}
		paths = append(paths, pathPatterns...)
	}

	var conditions []alb.Condition
	if len(hosts) != 0 {
		conditions = append(conditions, t.buildHostHeaderCondition(ctx, hosts))
	}
	if len(paths) != 0 {
		conditions = append(conditions, t.buildPathPatternCondition(ctx, paths))
	}
	conditionItems := make([]alb.Condition, 0)
	conditionStr := ing.Annotations[fmt.Sprintf(annotations.INGRESS_ALB_CONDITIONS_ANNOTATIONS, path.Backend.Service.Name)]
	if conditionStr != "" {
		klog.Infof("INGRESS_ALB_CONDITIONS_ANNOTATIONS: %s", conditionStr)
		err := json.Unmarshal([]byte(conditionStr), &conditionItems)
		if err != nil {
			return conditions, fmt.Errorf("buildRuleConditionsCommon: %s Unmarshal: %s", conditionStr, err.Error())
		}
		//for _, item := range conditionItems {
		conditions = append(conditions, conditionItems...)
		//}
	}

	if len(conditions) == 0 {
		conditions = append(conditions, t.buildPathPatternCondition(ctx, []string{"/*"}))
	}

	return conditions, nil
}
func (t *defaultModelBuildTask) buildFixedResponseAction(_ context.Context, actionCfg alb.Action) (*alb.Action, error) {
	if len(actionCfg.FixedResponseConfig.ContentType) == 0 {
		return nil, errors.New("missing FixedResponseConfig")
	}
	return &alb.Action{
		Type: util.RuleActionTypeFixedResponse,
		FixedResponseConfig: &alb.FixedResponseConfig{
			ContentType: actionCfg.FixedResponseConfig.ContentType,
			Content:     actionCfg.FixedResponseConfig.Content,
			HttpCode:    actionCfg.FixedResponseConfig.HttpCode,
		},
	}, nil
}

func (t *defaultModelBuildTask) buildRedirectAction(_ context.Context, actionCfg alb.Action) (*alb.Action, error) {
	if actionCfg.RedirectConfig == nil {
		return nil, errors.New("missing RedirectConfig")
	}
	return &alb.Action{
		Type: util.RuleActionTypeRedirect,
		RedirectConfig: &alb.RedirectConfig{
			Host:     actionCfg.RedirectConfig.Host,
			Path:     actionCfg.RedirectConfig.Path,
			Port:     actionCfg.RedirectConfig.Port,
			Protocol: actionCfg.RedirectConfig.Protocol,
			Query:    actionCfg.RedirectConfig.Query,
			HttpCode: actionCfg.RedirectConfig.HttpCode,
		},
	}, nil
}

func (t *defaultModelBuildTask) buildForwardAction(ctx context.Context, ing networking.Ingress, actionCfg alb.Action) (*alb.Action, error) {
	if actionCfg.ForwardConfig == nil {
		return nil, errors.New("missing ForwardConfig")
	}

	var serverGroupTuples []alb.ServerGroupTuple
	for _, sgp := range actionCfg.ForwardConfig.ServerGroups {
		svc := new(corev1.Service)
		svc.Namespace = ing.Namespace
		svc.Name = sgp.ServiceName
		modelSgp, err := t.buildServerGroup(ctx, &ing, svc, sgp.ServicePort)
		if err != nil {
			return nil, err
		}
		serverGroupTuples = append(serverGroupTuples, alb.ServerGroupTuple{
			ServerGroupID: modelSgp.ServerGroupID(),
			Weight:        sgp.Weight,
		})
	}

	return &alb.Action{
		Type: util.RuleActionTypeForward,
		ForwardConfig: &alb.ForwardActionConfig{
			ServerGroups: serverGroupTuples,
		},
	}, nil
}

func (t *defaultModelBuildTask) buildBackendAction(ctx context.Context, ing networking.Ingress, actionCfg alb.Action) (*alb.Action, error) {
	switch actionCfg.Type {
	case util.RuleActionTypeFixedResponse:
		return t.buildFixedResponseAction(ctx, actionCfg)
	case util.RuleActionTypeRedirect:
		return t.buildRedirectAction(ctx, actionCfg)
	case util.RuleActionTypeForward:
		return t.buildForwardAction(ctx, ing, actionCfg)
	}
	return nil, errors.Errorf("unknown action type: %v", actionCfg.Type)
}

func (t *defaultModelBuildTask) buildAction(ctx context.Context, ing networking.Ingress, action alb.Action) (alb.Action, error) {
	backendAction, err := t.buildBackendAction(ctx, ing, action)
	if err != nil {
		return alb.Action{}, err
	}
	return *backendAction, nil
}

func getIfOnlyWeight(ing *networking.Ingress) int {
	weight := 0
	w := annotations.GetStringAnnotationMutil(annotations.NginxCanaryWeight, annotations.AlbCanaryWeight, ing)
	header := annotations.GetStringAnnotationMutil(annotations.NginxCanaryByHeader, annotations.AlbCanaryByHeader, ing)
	headerValue := annotations.GetStringAnnotationMutil(annotations.NginxCanaryByHeaderValue, annotations.AlbCanaryByHeaderValue, ing)
	cookie := annotations.GetStringAnnotationMutil(annotations.NginxCanaryByCookie, annotations.AlbCanaryByCookie, ing)
	if header != "" || headerValue != "" || cookie != "" {
		return 0
	}
	weight, _ = strconv.Atoi(w)
	return weight
}

func (t *defaultModelBuildTask) buildRuleActions(ctx context.Context, ing *networking.Ingress, path *networking.HTTPIngressPath, canarySGP []alb.ServerGroupTuple, listen443 bool) ([]alb.Action, error) {
	actions := make([]alb.Action, 0)
	rawActions := make([]alb.Action, 0)
	var actionsMapArray []map[string]interface{}
	actionStr, exist := ing.Annotations[fmt.Sprintf(annotations.INGRESS_ALB_ACTIONS_ANNOTATIONS, path.Backend.Service.Name)]
	if exist {
		err := json.Unmarshal([]byte(actionStr), &actionsMapArray)
		if err != nil {
			klog.Errorf("buildRuleActions: %s Unmarshal: %s", actionStr, err.Error())
			return nil, err
		}
		for _, actionMap := range actionsMapArray {
			actionType, err := readActionType(actionMap["type"])
			if err != nil {
				return nil, err
			}
			actionCfg := actionCfgByType(actionType, actionMap)
			act, err := readAction(actionType, actionCfg)
			if err != nil {
				return nil, err
			}
			rawActions = append(rawActions, act)
		}
	}
	var extAction alb.Action
	if v, err := annotations.GetStringAnnotation(annotations.AlbRewriteTarget, ing); err == nil {
		extAction = alb.Action{
			Type: util.RuleActionTypeRewrite,
			RewriteConfig: &alb.RewriteConfig{
				Host:  "${host}",
				Path:  v,
				Query: "${query}",
			},
		}
	}
	rawActions = append(rawActions, extAction)

	var finalAction alb.Action
	var hasFinal bool = false
	var err error
	if v := annotations.GetStringAnnotationMutil(annotations.NginxSslRedirect, annotations.AlbSslRedirect, ing); v == "true" && !listen443 {
		finalAction = buildActionViaHostAndPath(ctx, path.Path)
		hasFinal = true
	} else if path.Backend.Service.Port.Name != "use-annotation" {
		canary := annotations.GetStringAnnotationMutil(annotations.NginxCanary, annotations.AlbCanary, ing)
		finalAction, err = t.buildActionForwardSGP(ctx, ing, path, canary != "true" && len(canarySGP) > 0, canarySGP)
		hasFinal = true
	}
	if err != nil {
		return nil, err
	}
	if hasFinal {
		rawActions = append(rawActions, finalAction)
	}
	// buildExtAction
	for _, act := range rawActions {
		if act.Type == util.RuleActionTypeInsertHeader ||
			act.Type == util.RuleActionTypeRewrite {
			actions = append(actions, act)
		}
	}
	// buildFinalAction
	var finalTypes []string
	for _, act := range rawActions {
		if act.Type == util.RuleActionTypeForward ||
			act.Type == util.RuleActionTypeRedirect ||
			act.Type == util.RuleActionTypeFixedResponse {
			actions = append(actions, act)
			finalTypes = append(finalTypes, act.Type)
		}
	}
	if len(finalTypes) > 1 {
		return actions, fmt.Errorf("multi finalType action find: %v", finalTypes)
	}
	return actions, nil
}

func (t *defaultModelBuildTask) buildActionForwardSGP(ctx context.Context, ing *networking.Ingress, path *networking.HTTPIngressPath, withCanary bool, canarySGP []alb.ServerGroupTuple) (alb.Action, error) {
	action := buildActionViaServiceAndServicePort(ctx, path.Backend.Service.Name, int(path.Backend.Service.Port.Number), 100)
	if withCanary {
		canaryWeight := 0
		for _, cw := range canarySGP {
			canaryWeight += cw.Weight
		}
		if canaryWeight == 100 {
			action.ForwardConfig.ServerGroups = canarySGP
		} else {
			for i := range action.ForwardConfig.ServerGroups {
				action.ForwardConfig.ServerGroups[i].Weight = 100 - canaryWeight
			}
			action.ForwardConfig.ServerGroups = append(action.ForwardConfig.ServerGroups, canarySGP...)
		}
	}
	var serverGroupTuples []alb.ServerGroupTuple
	for _, sgp := range action.ForwardConfig.ServerGroups {
		svc := new(corev1.Service)
		svc.Namespace = ing.Namespace
		svc.Name = sgp.ServiceName
		modelSgp, err := t.buildServerGroup(ctx, ing, svc, sgp.ServicePort)
		if err != nil {
			return alb.Action{}, err
		}
		serverGroupTuples = append(serverGroupTuples, alb.ServerGroupTuple{
			ServerGroupID: modelSgp.ServerGroupID(),
			Weight:        sgp.Weight,
		})
	}
	action.ForwardConfig.ServerGroups = serverGroupTuples
	return action, nil
}

func readActionType(actType interface{}) (string, error) {
	tp := actType.(string)
	if tp != util.RuleActionTypeFixedResponse &&
		tp != util.RuleActionTypeRedirect &&
		tp != util.RuleActionTypeInsertHeader {
		return tp, fmt.Errorf("readActionType Failed(unknown action type): %s", tp)
	}
	return tp, nil
}

func actionCfgByType(actType string, actMap map[string]interface{}) interface{} {
	return actMap[actionTypeMap[actType]]
}

func readAction(actType string, actCfg interface{}) (alb.Action, error) {
	toAct := alb.Action{
		RedirectConfig: &alb.RedirectConfig{
			Host:     "${host}",
			HttpCode: "301",
			Path:     "${path}",
			Port:     "${port}",
			Protocol: "${protocol}",
			Query:    "${query}",
		},
	}
	if actCfg == nil {
		return toAct, fmt.Errorf("action Config is nil with type(%s)", actType)
	}
	bActCfg, err := json.Marshal(actCfg)
	if err != nil {
		return toAct, err
	}
	switch actType {
	case util.RuleActionTypeFixedResponse:
		toAct.Type = util.RuleActionTypeFixedResponse
		err = json.Unmarshal(bActCfg, &toAct.FixedResponseConfig)
	case util.RuleActionTypeRedirect:
		toAct.Type = util.RuleActionTypeRedirect
		err = json.Unmarshal(bActCfg, &toAct.RedirectConfig)
	case util.RuleActionTypeInsertHeader:
		toAct.Type = util.RuleActionTypeInsertHeader
		err = json.Unmarshal(bActCfg, &toAct.InsertHeaderConfig)
	default:
		return toAct, fmt.Errorf("readAction Failed(unknown action type): %s", actType)
	}
	return toAct, err
}
