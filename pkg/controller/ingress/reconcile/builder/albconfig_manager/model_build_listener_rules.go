package albconfigmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/store"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb/configcache"

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
)

var (
	lowerRuleActionTypeFixedResponse = strings.ToLower(util.RuleActionTypeFixedResponse)
	lowerRuleActionTypeRedirect      = strings.ToLower(util.RuleActionTypeRedirect)
	lowerRuleActionTypeInsertHeader  = strings.ToLower(util.RuleActionTypeInsertHeader)
	lowerRuleActionTypeTrafficMirror = strings.ToLower(util.RuleActionTypeTrafficMirror)
	lowerRuleActionTypeRemoveHeader  = strings.ToLower(util.RuleActionTypeRemoveHeader)
	lowerRuleActionTypeForward       = strings.ToLower(util.RuleActionTypeForward)
	lowerRuleActionTypeRewrite       = strings.ToLower(util.RuleActionTypeRewrite)
	lowerRuleActionTypeTrafficLimit  = strings.ToLower(util.RuleActionTypeTrafficLimit)

	lowerRuleConditionFieldHost          = strings.ToLower(util.RuleConditionFieldHost)
	lowerRuleConditionFieldPath          = strings.ToLower(util.RuleConditionFieldPath)
	lowerRuleConditionFieldHeader        = strings.ToLower(util.RuleConditionFieldHeader)
	lowerRuleConditionFieldQueryString   = strings.ToLower(util.RuleConditionFieldQueryString)
	lowerRuleConditionFieldMethod        = strings.ToLower(util.RuleConditionFieldMethod)
	lowerRuleConditionFieldCookie        = strings.ToLower(util.RuleConditionFieldCookie)
	lowerRuleConditionFieldSourceIp      = strings.ToLower(util.RuleConditionFieldSourceIp)
	lowerRuleConditionResponseHeader     = strings.ToLower(util.RuleConditionResponseHeader)
	lowerRuleConditionResponseStatusCode = strings.ToLower(util.RuleConditionResponseStatusCode)
)

func (t *defaultModelBuildTask) checkCanaryAndCustomizeCondition(ing networking.Ingress) (bool, error) {
	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			_, exist := ing.Annotations[fmt.Sprintf(annotations.INGRESS_ALB_CONDITIONS_ANNOTATIONS, path.Backend.Service.Name)]
			if exist {
				klog.Errorf("%v can't exist Canary and customize condition at the same time", util.NamespacedName(&ing))
				return false, fmt.Errorf("%v can't exist Canary and customize condition at the same time", util.NamespacedName(&ing))
			}
		}
	}
	return true, nil
}

type canarySGPWithIngress struct {
	canaryServerGroupTuple alb.ServerGroupTuple
	canaryIngress          networking.Ingress
}

func (t *defaultModelBuildTask) buildListenerRules(ctx context.Context, lsID core.StringToken, port int32, protocol Protocol, ingList []networking.Ingress) error {
	if len(ingList) > 0 {
		ing := ingList[0]
		if _, ok := ing.Labels[util.KnativeIngress]; ok {
			oldVersion := false
			for k, actionStr := range ing.Annotations {
				if strings.HasPrefix(k, "alb.ingress.kubernetes.io/actions") && !strings.HasPrefix(actionStr, "[") {
					oldVersion = true
					break
				}
			}
			if oldVersion {
				return t.buildListenerRulesCommon(ctx, lsID, port, ingList)
			}
		}
	}
	var rules []alb.ListenerRule
	canaryServerGroupWithIngress := make(map[string][]canarySGPWithIngress, 0)
	nonCanaryPath := make(map[string]bool, 0)
	for _, ing := range ingList {
		if v := annotations.GetStringAnnotationMutil(annotations.NginxCanary, annotations.AlbCanary, &ing); v == "true" {
			//canary and customize condition can't simultaneously exist
			if ok, err := t.checkCanaryAndCustomizeCondition(ing); !ok {
				t.errResultWithIngress[&ing] = err
				return err
			}
			weight := getIfOnlyWeight(&ing)
			if weight == 0 {
				continue
			}
			checkCanaryWeightAndForwardAction, _ := store.CheckAnnotationForwardAction(ing)
			if checkCanaryWeightAndForwardAction {
				err := fmt.Errorf("%v can't exist CanaryWeight and annotation forwardAction at the same time", util.NamespacedName(&ing))
				t.errResultWithIngress[&ing] = err
				return err
			}
			for _, rule := range ing.Spec.Rules {
				if rule.HTTP == nil {
					continue
				}
				for _, path := range rule.HTTP.Paths {
					if _, ok := canaryServerGroupWithIngress[rule.Host+"-"+path.Path]; !ok {
						canaryServerGroupWithIngress[rule.Host+"-"+path.Path] = make([]canarySGPWithIngress, 0)
					}
					canaryServerGroupWithIngress[rule.Host+"-"+path.Path] = append(canaryServerGroupWithIngress[rule.Host+"-"+path.Path], canarySGPWithIngress{
						canaryServerGroupTuple: alb.ServerGroupTuple{
							ServiceName: path.Backend.Service.Name,
							ServicePort: int(path.Backend.Service.Port.Number),
							Weight:      weight,
						},
						canaryIngress: ing,
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
			if rule.Host != "" && !forceListenIngress(ing) {
				isTlsRule := withTlsRule(protocol, rule.Host, ing)
				if protocol == ProtocolHTTPS && !isTlsRule {
					continue
				}
				if protocol == ProtocolHTTP && isTlsRule {
					continue
				}
			}
			for _, path := range rule.HTTP.Paths {
				if _, ok := nonCanaryPath[rule.Host+"-"+path.Path]; !ok {
					continue
				}
				actions, err := t.buildRuleActions(ctx, &ing, &path, canaryServerGroupWithIngress[rule.Host+"-"+path.Path], port == 443)
				if err != nil {
					t.errResultWithIngress[&ing] = err
					return errors.Wrapf(err, "buildListenerRules-Actions(ingress: %v)", util.NamespacedName(&ing))
				}
				conditions, err := t.buildRuleConditions(ctx, rule, path, ing)
				if err != nil {
					t.errResultWithIngress[&ing] = err
					return errors.Wrapf(err, "buildListenerRules-Conditions(ingress: %v)", util.NamespacedName(&ing))
				}
				direction, err := t.buildRuleDirection(ctx, rule, path, ing)
				if err != nil {
					t.errResultWithIngress[&ing] = err
					return errors.Wrapf(err, "buildListenerRules-Direction(ingress: %v)", util.NamespacedName(&ing))
				}
				rules = append(rules, alb.ListenerRule{
					Spec: alb.ListenerRuleSpec{
						ListenerID: lsID,
						ALBListenerRuleSpec: alb.ALBListenerRuleSpec{
							RuleActions:    actions,
							RuleConditions: conditions,
							RuleDirection:  direction,
						},
					},
				})
			}
		}
	}

	priority := 1
	for _, rule := range rules {
		ruleResID := fmt.Sprintf("%v-%v:%v", port, protocol, priority)
		klog.Infof("ruleResID: %s", ruleResID)
		lrs := alb.ListenerRuleSpec{
			ListenerID: lsID,
			ALBListenerRuleSpec: alb.ALBListenerRuleSpec{
				Priority:       priority,
				RuleConditions: rule.Spec.RuleConditions,
				RuleActions:    rule.Spec.RuleActions,
				RuleName:       fmt.Sprintf("%v-%v-%v", ListenerRuleNamePrefix, port, priority),
				RuleDirection:  rule.Spec.RuleDirection,
			},
		}
		_ = alb.NewListenerRule(t.stack, ruleResID, lrs)
		priority += 1
	}

	return nil
}

/*
 * true if rule need config on https listener
 */
func withTlsRule(protocol Protocol, host string, ing networking.Ingress) bool {
	// only https listen && len(tls) == 0: use annotation listen-port config https
	if protocol == ProtocolHTTPS && len(ing.Spec.TLS) == 0 {
		return true
	}
	// host == "": the rule use to default request
	if host == "" {
		return true
	}
	for _, tls := range ing.Spec.TLS {
		if contains(tls.Hosts, host) {
			return true
		}
	}
	return false
}

func forceListenIngress(ing networking.Ingress) bool {
	if v := annotations.GetStringAnnotationMutil(annotations.NginxSslRedirect, annotations.AlbSslRedirect, &ing); v == "true" {
		return true
	}
	_, err := annotations.GetStringAnnotation(annotations.ListenPorts, &ing)
	return err == nil
}

func contains(s []string, e string) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

func (t *defaultModelBuildTask) buildListenerRulesCommon(ctx context.Context, lsID core.StringToken, port int32, ingList []networking.Ingress) error {
	var rules []alb.ListenerRule
	for _, ing := range ingList {
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				actions := make([]alb.Action, 0)
				actionStr := ing.Annotations[fmt.Sprintf(annotations.INGRESS_ALB_ACTIONS_ANNOTATIONS, path.Backend.Service.Name)]
				if actionStr != "" {
					var action alb.Action
					err := json.Unmarshal([]byte(actionStr), &action)
					if err != nil {
						klog.Errorf("buildListenerRulesCommon: %s Unmarshal: %s", actionStr, err.Error())
						continue
					}
					action2, err := t.buildAction(ctx, ing, action)
					if err != nil {
						klog.Errorf("buildListenerRulesCommon error: %s", err.Error())
						continue
					}
					actions = append(actions, action2)
					if v, _ := annotations.GetStringAnnotation(annotations.AlbSslRedirect, &ing); v == "true" && !(port == 443) {
						actions = []alb.Action{buildActionViaHostAndPath(ctx, path.Path)}
					}
				}
				klog.Infof("INGRESS_ALB_ACTIONS_ANNOTATIONS: %s", actionStr)
				conditions, err := t.buildRuleConditionsCommon(ctx, rule, path, ing)
				if err != nil {
					klog.Errorf("buildListenerRulesCommon error: %s", err.Error())
					continue
				}
				lrs := alb.ListenerRuleSpec{
					ListenerID: lsID,
				}
				lrs.RuleActions = actions
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
	if v, err := annotations.GetStringAnnotation(annotations.AlbUseRegexPath, &ing); err == nil && v != path {
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

func (t *defaultModelBuildTask) buildSourceIpCondition(_ context.Context, sourceIps []string) alb.Condition {
	return alb.Condition{
		Type: util.RuleConditionFieldSourceIp,
		SourceIpConfig: alb.SourceIpConfig{
			Values: sourceIps,
		},
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

func (t *defaultModelBuildTask) buildMethodCondition(_ context.Context, methods []string) alb.Condition {
	return alb.Condition{
		Type: util.RuleConditionFieldMethod,
		MethodConfig: alb.MethodConfig{
			Values: methods,
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

func (t *defaultModelBuildTask) buildResponseStatusCodeCondition(_ context.Context, statusCodes []string) alb.Condition {
	return alb.Condition{
		Type: util.RuleConditionResponseStatusCode,
		ResponseStatusCodeConfig: alb.ResponseStatusCodeConfig{
			Values: statusCodes,
		},
	}
}

func (t *defaultModelBuildTask) buildRuleDirection(ctx context.Context, rule networking.IngressRule,
	path networking.HTTPIngressPath, ing networking.Ingress) (string, error) {
	direction, exist := ing.Annotations[fmt.Sprintf(annotations.INGRESS_ALB_RULE_DIRECTION, path.Backend.Service.Name)]
	if exist {
		switch direction {
		case util.RuleRequestDirection:
			return util.RuleRequestDirection, nil
		case util.RuleResponseDirection:
			return util.RuleResponseDirection, nil
		default:
			return "", fmt.Errorf("readDirection Failed(unknown direction type): %s", direction)
		}
	}
	return util.RuleRequestDirection, nil
}

func (t *defaultModelBuildTask) buildRuleConditions(ctx context.Context, rule networking.IngressRule,
	path networking.HTTPIngressPath, ing networking.Ingress) ([]alb.Condition, error) {
	var conditions []alb.Condition
	var hosts []string
	var paths []string
	var sourceIps []string
	var methods []string
	var statusCodes []string
	conditionStr, exist := ing.Annotations[fmt.Sprintf(annotations.INGRESS_ALB_CONDITIONS_ANNOTATIONS, path.Backend.Service.Name)]
	if exist && conditionStr != "" {
		conditionConfig := []configcache.Condition{}
		err := json.Unmarshal([]byte(conditionStr), &conditionConfig)
		if err != nil {
			klog.Errorf("buildRuleConditions: %s Unmarshal: %s", conditionStr, err.Error())
			return nil, err
		}
		for _, cond := range conditionConfig {
			switch strings.ToLower(cond.Type) {
			case lowerRuleConditionFieldHost:
				hosts = append(hosts, cond.HostConfig.Values...)
			case lowerRuleConditionFieldPath:
				paths = append(paths, cond.PathConfig.Values...)
			case lowerRuleConditionFieldMethod:
				methods = append(methods, cond.MethodConfig.Values...)
			case lowerRuleConditionFieldSourceIp:
				sourceIps = append(sourceIps, cond.SourceIpConfig.Values...)
			case lowerRuleConditionResponseStatusCode:
				statusCodes = append(statusCodes, cond.ResponseStatusCodeConfig.Values...)
			case lowerRuleConditionFieldHeader:
				headerCondition := alb.Condition{
					Type: util.RuleConditionFieldHeader,
					HeaderConfig: alb.HeaderConfig{
						Key:    cond.HeaderConfig.Key,
						Values: cond.HeaderConfig.Values,
					},
				}
				conditions = append(conditions, headerCondition)
			case lowerRuleConditionFieldQueryString:
				queryValues := make([]alb.Value, 0)
				for _, value := range cond.QueryStringConfig.Values {
					queryValues = append(queryValues, alb.Value{
						Key:   value.Key,
						Value: value.Value,
					})
				}
				queryStringCondition := alb.Condition{
					Type: util.RuleConditionFieldQueryString,
					QueryStringConfig: alb.QueryStringConfig{
						Values: queryValues,
					},
				}
				conditions = append(conditions, queryStringCondition)
			case lowerRuleConditionFieldCookie:
				cookieValues := make([]alb.Value, 0)
				for _, value := range cond.CookieConfig.Values {
					cookieValues = append(cookieValues, alb.Value{
						Key:   value.Key,
						Value: value.Value,
					})
				}
				cookieCondition := alb.Condition{
					Type: util.RuleConditionFieldCookie,
					CookieConfig: alb.CookieConfig{
						Values: cookieValues,
					},
				}
				conditions = append(conditions, cookieCondition)
			case lowerRuleConditionResponseHeader:
				responseHeaderCondition := alb.Condition{
					Type: util.RuleConditionResponseHeader,
					ResponseHeaderConfig: alb.ResponseHeaderConfig{
						Key:    cond.ResponseHeaderConfig.Key,
						Values: cond.ResponseHeaderConfig.Values,
					},
				}
				conditions = append(conditions, responseHeaderCondition)
			default:
				return nil, fmt.Errorf("readCondition Failed(unknown condition type): %s", cond.Type)
			}
		}
	}

	if rule.Host != "" {
		hosts = append(hosts, rule.Host)
	}
	if path.Path != "" {
		pathPatterns, err := t.buildPathPatterns(path.Path, path.PathType, ing)
		if err != nil {
			return nil, err
		}
		paths = append(paths, pathPatterns...)
	}

	if len(hosts) != 0 {
		conditions = append(conditions, t.buildHostHeaderCondition(ctx, hosts))
	}
	if len(paths) != 0 {
		conditions = append(conditions, t.buildPathPatternCondition(ctx, paths))
	}
	if len(methods) != 0 {
		conditions = append(conditions, t.buildMethodCondition(ctx, methods))
	}
	if len(sourceIps) != 0 {
		conditions = append(conditions, t.buildSourceIpCondition(ctx, sourceIps))
	}
	if len(statusCodes) != 0 {
		conditions = append(conditions, t.buildResponseStatusCodeCondition(ctx, statusCodes))
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

func (t *defaultModelBuildTask) buildQpsLimitAction(ctx context.Context, qps, qpsPerIp string, ing *networking.Ingress) (*alb.Action, error) {
	if qps != "" && qpsPerIp == "" {
		qpsi, err := strconv.Atoi(qps)
		if err != nil {
			return nil, err
		}
		if qpsi < util.ActionTrafficLimitQpsMin {
			return nil, fmt.Errorf("traffic limit action qps out of range: %d", qpsi)
		}
		return &alb.Action{
			Type: util.RuleActionTypeTrafficLimit,
			TrafficLimitConfig: &alb.TrafficLimitConfig{
				QPS: qpsi,
			},
		}, nil
	}
	if qps == "" && qpsPerIp != "" {
		qpsPerIpi, err := strconv.Atoi(qpsPerIp)
		if err != nil {
			return nil, err
		}
		if qpsPerIpi < util.ActionTrafficLimitQpsMin {
			return nil, fmt.Errorf("traffic limit action qps per ip out of range: %d", qpsPerIpi)
		}
		return &alb.Action{
			Type: util.RuleActionTypeTrafficLimit,
			TrafficLimitConfig: &alb.TrafficLimitConfig{
				PerIpQps: qpsPerIpi,
			},
		}, nil
	}
	if qps != "" && qpsPerIp != "" {
		qpsi, err := strconv.Atoi(qps)
		if err != nil {
			return nil, err
		}
		if qpsi < util.ActionTrafficLimitQpsMin {
			return nil, fmt.Errorf("traffic limit action qps out of range: %d", qpsi)
		}
		qpsPerIpi, err := strconv.Atoi(qpsPerIp)
		if err != nil {
			return nil, err
		}
		if qpsPerIpi < util.ActionTrafficLimitQpsMin {
			return nil, fmt.Errorf("traffic limit action qps per ip out of range: %d", qpsPerIpi)
		}
		if qpsPerIpi >= qpsi {
			return nil, fmt.Errorf("traffic limit action qps per ip(%d)must less than qps(%d)", qpsPerIpi, qpsi)
		}
		return &alb.Action{
			Type: util.RuleActionTypeTrafficLimit,
			TrafficLimitConfig: &alb.TrafficLimitConfig{
				QPS:      qpsi,
				PerIpQps: qpsPerIpi,
			},
		}, nil
	}
	return nil, fmt.Errorf("invalid parameter for qps limit qps=%s, qpsPerIp=%s", qps, qpsPerIp)
}

func (t *defaultModelBuildTask) buildRuleActions(ctx context.Context, ing *networking.Ingress, path *networking.HTTPIngressPath, canaryWithIngress []canarySGPWithIngress, listen443 bool) ([]alb.Action, error) {
	actions := make([]alb.Action, 0)
	rawActions := make([]alb.Action, 0)
	sslRedirectActions := make([]alb.Action, 0)
	var extAction alb.Action
	qps, _ := annotations.GetStringAnnotation(annotations.AlbTrafficLimitQps, ing)
	qpsPerIp, _ := annotations.GetStringAnnotation(annotations.AlbTrafficLimitIpQps, ing)
	if qps != "" || qpsPerIp != "" {
		limitAction, err := t.buildQpsLimitAction(ctx, qps, qpsPerIp, ing)
		if err != nil {
			return nil, err
		}
		rawActions = append(rawActions, *limitAction)
	}
	// CorsConfig must after TrafficMirror and before other
	if v, _ := annotations.GetStringAnnotation(annotations.AlbEnableCors, ing); v == "true" {
		var corsAllowOrigin []string = splitAndTrim(util.DefaultCorsAllowOrigin)
		if v, err := annotations.GetStringAnnotation(annotations.AlbCorsAllowOrigin, ing); err == nil {
			corsAllowOrigin = splitAndTrim(v)
		}
		var corsAllowMethods []string = splitAndTrim(util.DefaultCorsAllowMethods)
		if v, err := annotations.GetStringAnnotation(annotations.AlbCorsAllowMethods, ing); err == nil {
			corsAllowMethods = splitAndTrim(v)
		}
		var corsAllowHeaders []string = splitAndTrim(util.DefaultCorsAllowHeaders)
		if v, err := annotations.GetStringAnnotation(annotations.AlbCorsAllowHeaders, ing); err == nil {
			corsAllowHeaders = splitAndTrim(v)
		}
		var corsExposeHeaders []string
		if v, err := annotations.GetStringAnnotation(annotations.AlbCorsExposeHeaders, ing); err == nil {
			corsExposeHeaders = splitAndTrim(v)
		}
		var corsAllowCredentials string = util.DefaultCorsAllowCredentials
		if v, err := annotations.GetStringAnnotation(annotations.AlbCorsAllowCredentials, ing); err == nil {
			if v == "true" || v == "false" {
				corsAllowCredentials = map[string]string{
					"true":  "on",
					"false": "off",
				}[v]
			} else {
				klog.Warning("Unexpect AlbCorsAllowCredentials value, expect: true or false, got " + v)
			}
		}
		var corsMaxAge string = util.DefaultCorsMaxAge
		if v, err := annotations.GetStringAnnotation(annotations.AlbCorsMaxAge, ing); err == nil {
			corsMaxAge = v
		}

		corsAction := alb.Action{
			Type: util.RuleActionTypeCors,
			CorsConfig: &alb.CorsConfig{
				AllowCredentials: corsAllowCredentials,
				MaxAge:           corsMaxAge,
				AllowOrigin:      corsAllowOrigin,
				AllowMethods:     corsAllowMethods,
				AllowHeaders:     corsAllowHeaders,
				ExposeHeaders:    corsExposeHeaders,
			},
		}
		rawActions = append(rawActions, corsAction)
	}
	actionStr, exist := ing.Annotations[fmt.Sprintf(annotations.INGRESS_ALB_ACTIONS_ANNOTATIONS, path.Backend.Service.Name)]
	if exist {
		actionsArray := make([]configcache.Action, 0)
		err := json.Unmarshal([]byte(actionStr), &actionsArray)
		if err != nil {
			klog.Errorf("buildRuleActions: %s Unmarshal: %s", actionStr, err.Error())
			return nil, err
		}
		for _, action := range actionsArray {
			if qps != "" {
				if strings.EqualFold(action.Type, util.RuleActionTypeTrafficLimit) {
					return nil, fmt.Errorf(" %v can't exist action trafficlimit and annotation traffic-limit-qps at the same time", util.NamespacedName(ing))
				}
			}
			if qpsPerIp != "" {
				if strings.EqualFold(action.Type, util.RuleActionTypeTrafficLimit) {
					return nil, fmt.Errorf(" %v can't exist action trafficlimit and annotation traffic-limit-ip-qps at the same time", util.NamespacedName(ing))
				}
			}
			act, err := t.toCustomAction(ctx, ing, action)
			if err != nil {
				return nil, err
			}
			rawActions = append(rawActions, act)
		}
	}
	if v, err := annotations.GetStringAnnotation(annotations.AlbRewriteTarget, ing); err == nil {
		extAction = alb.Action{
			Type: util.RuleActionTypeRewrite,
			RewriteConfig: &alb.RewriteConfig{
				Host:  "${host}",
				Path:  v,
				Query: "${query}",
			},
		}
		rawActions = append(rawActions, extAction)
	}
	var finalAction alb.Action
	var hasFinal bool = false
	var err error
	if v := annotations.GetStringAnnotationMutil(annotations.NginxSslRedirect, annotations.AlbSslRedirect, ing); v == "true" && !listen443 {
		for _, act := range rawActions {
			if act.Type == util.RuleActionTypeInsertHeader ||
				act.Type == util.RuleActionTypeRemoveHeader ||
				act.Type == util.RuleActionTypeTrafficLimit ||
				act.Type == util.RuleActionTypeCors {
				sslRedirectActions = append(sslRedirectActions, act)
			}
		}
		sslRedirectActions = append(sslRedirectActions, buildActionViaHostAndPath(ctx, path.Path))
		return sslRedirectActions, nil
	} else if path.Backend.Service.Port.Name != "use-annotation" {
		canary := annotations.GetStringAnnotationMutil(annotations.NginxCanary, annotations.AlbCanary, ing)
		finalAction, err = t.buildActionForwardSGP(ctx, ing, path, canary != "true" && len(canaryWithIngress) > 0, canaryWithIngress)
		hasFinal = true
	}
	if err != nil {
		return nil, err
	}
	if hasFinal {
		rawActions = append(rawActions, finalAction)
	}
	// buildTrafficLimitAction
	for _, act := range rawActions {
		if act.Type == util.RuleActionTypeTrafficLimit {
			actions = append(actions, act)
		}
	}
	// buildExtAction
	for _, act := range rawActions {
		if act.Type == util.RuleActionTypeInsertHeader ||
			act.Type == util.RuleActionTypeRemoveHeader ||
			act.Type == util.RuleActionTypeRewrite ||
			act.Type == util.RuleActionTypeCors ||
			act.Type == util.RuleActionTypeTrafficMirror {
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

func (t *defaultModelBuildTask) buildForwardCanarySGP(ctx context.Context, canaryServerGroupWithIngress []canarySGPWithIngress) ([]alb.ServerGroupTuple, error) {
	var serverGroupTuples []alb.ServerGroupTuple
	for _, sgpWithIngress := range canaryServerGroupWithIngress {
		svc := new(corev1.Service)
		svc.Namespace = sgpWithIngress.canaryIngress.Namespace
		svc.Name = sgpWithIngress.canaryServerGroupTuple.ServiceName
		modelSgp, err := t.buildServerGroup(ctx, &sgpWithIngress.canaryIngress, svc, sgpWithIngress.canaryServerGroupTuple.ServicePort)
		if err != nil {
			return serverGroupTuples, err
		}
		serverGroupTuples = append(serverGroupTuples, alb.ServerGroupTuple{
			ServerGroupID: modelSgp.ServerGroupID(),
			Weight:        sgpWithIngress.canaryServerGroupTuple.Weight,
		})
	}
	return serverGroupTuples, nil
}

func (t *defaultModelBuildTask) buildActionForwardSGP(ctx context.Context, ing *networking.Ingress, path *networking.HTTPIngressPath, withCanary bool, canaryServerGroupWithIngress []canarySGPWithIngress) (alb.Action, error) {
	action := buildActionViaServiceAndServicePort(ctx, path.Backend.Service.Name, int(path.Backend.Service.Port.Number), 100)
	var canaryServerGroupTuples []alb.ServerGroupTuple
	if withCanary {
		canaryWeight := 0
		for _, canaryWithIngress := range canaryServerGroupWithIngress {
			canaryWeight += canaryWithIngress.canaryServerGroupTuple.Weight
		}
		if canaryWeight == 100 {
			serverGroupTuples, err := t.buildForwardCanarySGP(ctx, canaryServerGroupWithIngress)
			if err != nil {
				return alb.Action{}, err
			}
			action.ForwardConfig.ServerGroups = serverGroupTuples
			return action, nil
		} else {
			for i := range action.ForwardConfig.ServerGroups {
				action.ForwardConfig.ServerGroups[i].Weight = 100 - canaryWeight
			}
			serverGroupTuples, err := t.buildForwardCanarySGP(ctx, canaryServerGroupWithIngress)
			if err != nil {
				return alb.Action{}, err
			}
			canaryServerGroupTuples = serverGroupTuples
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
	serverGroupTuples = append(serverGroupTuples, canaryServerGroupTuples...)
	action.ForwardConfig.ServerGroups = serverGroupTuples
	return action, nil
}

func (t *defaultModelBuildTask) toCustomAction(ctx context.Context, ing *networking.Ingress, action configcache.Action) (alb.Action, error) {
	actType := strings.ToLower(action.Type)
	var toAct *alb.Action
	var err error
	switch actType {
	case lowerRuleActionTypeFixedResponse:
		toAct = &alb.Action{
			Type: util.RuleActionTypeFixedResponse,
			FixedResponseConfig: &alb.FixedResponseConfig{
				Content:     action.FixedResponseConfig.Content,
				ContentType: action.FixedResponseConfig.ContentType,
				HttpCode:    action.FixedResponseConfig.HttpCode,
			},
		}
	case lowerRuleActionTypeRedirect:
		toAct = &alb.Action{
			Type: util.RuleActionTypeRedirect,
			RedirectConfig: &alb.RedirectConfig{
				Host:     "${host}",
				HttpCode: "301",
				Path:     "${path}",
				Port:     "${port}",
				Protocol: "${protocol}",
				Query:    "${query}",
			},
		}
		bActCfg, _ := json.Marshal(action.RedirectConfig)
		err = json.Unmarshal(bActCfg, &toAct.RedirectConfig)
	case lowerRuleActionTypeInsertHeader:
		toAct = &alb.Action{
			Type: util.RuleActionTypeInsertHeader,
			InsertHeaderConfig: &alb.InsertHeaderConfig{
				CoverEnabled: action.InsertHeaderConfig.CoverEnabled,
				Key:          action.InsertHeaderConfig.Key,
				Value:        action.InsertHeaderConfig.Value,
				ValueType:    action.InsertHeaderConfig.ValueType,
			},
		}
	case lowerRuleActionTypeTrafficMirror:
		trafficMirrorAction := buildTrafficMirrorAction(action)
		toAct = &trafficMirrorAction
	case lowerRuleActionTypeRemoveHeader:
		toAct = &alb.Action{
			Type: util.RuleActionTypeRemoveHeader,
			RemoveHeaderConfig: &alb.RemoveHeaderConfig{
				Key: action.RemoveHeaderConfig.Key,
			},
		}
	case lowerRuleActionTypeForward:
		forwardAction, aErr := t.buildAnnotationForwardAction(ctx, ing, action)
		if aErr != nil {
			err = fmt.Errorf("build ForwardAction Failed: %v", aErr)
		}
		toAct = &forwardAction
	case lowerRuleActionTypeRewrite:
		toAct = &alb.Action{
			Type: util.RuleActionTypeRewrite,
			RewriteConfig: &alb.RewriteConfig{
				Host:  action.RewriteConfig.Host,
				Path:  action.RewriteConfig.Path,
				Query: action.RewriteConfig.Query,
			},
		}
	case lowerRuleActionTypeTrafficLimit:
		QpsLimitAction, aErr := t.buildQpsLimitAction(ctx, action.TrafficLimitConfig.QPS, action.TrafficLimitConfig.QPSPerIp, ing)
		if aErr != nil {
			err = fmt.Errorf("build TrafficLimitAction Failed: %v", aErr)
		} else {
			toAct = QpsLimitAction
		}
	default:
		err = fmt.Errorf("readAction Failed(unknown action type): %s", actType)
	}
	return *toAct, err
}

func buildTrafficMirrorAction(action configcache.Action) alb.Action {
	var trafficMirrorSgpTuples []alb.TrafficMirrorServerGroupTuple
	for _, sgp := range action.TrafficMirrorConfig.MirrorGroupConfig.ServerGroupTuples {
		trafficMirrorSgpTuples = append(trafficMirrorSgpTuples, alb.TrafficMirrorServerGroupTuple{
			ServerGroupID: sgp.ServerGroupID,
			ServiceName:   sgp.ServiceName,
			ServicePort:   sgp.ServicePort,
			Weight:        sgp.Weight,
		})
	}
	return alb.Action{
		Type: util.RuleActionTypeTrafficMirror,
		TrafficMirrorConfig: &alb.TrafficMirrorConfig{
			TargetType: action.TrafficMirrorConfig.TargetType,
			MirrorGroupConfig: alb.MirrorGroupConfig{
				ServerGroupTuples: trafficMirrorSgpTuples,
			},
		},
	}
}

func (t *defaultModelBuildTask) buildAnnotationForwardAction(ctx context.Context, ing *networking.Ingress, action configcache.Action) (alb.Action, error) {
	var forwardSgpTuples []alb.ServerGroupTuple
	for _, sgp := range action.ForwardConfig.ServerGroups {
		var sgpID core.StringToken
		if sgp.ServerGroupID != "" {
			sgpID = core.LiteralStringToken(sgp.ServerGroupID)
			forwardSgpTuples = append(forwardSgpTuples, alb.ServerGroupTuple{
				ServerGroupID: sgpID,
				Weight:        sgp.Weight,
			})
		} else {
			svc := new(corev1.Service)
			svc.Namespace = ing.Namespace
			svc.Name = sgp.ServiceName
			modelSgp, err := t.buildServerGroup(ctx, ing, svc, sgp.ServicePort)
			if err != nil {
				return alb.Action{}, err
			}

			forwardSgpTuples = append(forwardSgpTuples, alb.ServerGroupTuple{
				ServerGroupID: modelSgp.ServerGroupID(),
				ServiceName:   sgp.ServiceName,
				ServicePort:   sgp.ServicePort,
				Weight:        sgp.Weight,
			})
		}
	}
	return alb.Action{
		Type: util.RuleActionTypeForward,
		ForwardConfig: &alb.ForwardActionConfig{
			ServerGroups: forwardSgpTuples,
		},
	}, nil
}

func splitAndTrim(values string) []string {
	ret := make([]string, 0)
	for _, value := range strings.Split(values, ",") {
		ret = append(ret, strings.Trim(value, " "))
	}
	return ret
}
