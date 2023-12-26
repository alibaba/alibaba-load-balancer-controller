package alb

import (
	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb/core"
)

var _ core.Resource = &ListenerRule{}

type ListenerRule struct {
	core.ResourceMeta `json:"-"`

	Spec ListenerRuleSpec `json:"spec"`

	Status *ListenerRuleStatus `json:"status,omitempty"`
}

func NewListenerRule(stack core.Manager, id string, spec ListenerRuleSpec) *ListenerRule {
	lr := &ListenerRule{
		ResourceMeta: core.NewResourceMeta(stack, "ALIYUN::ALB::RULE", id),
		Spec:         spec,
		Status:       nil,
	}
	_ = stack.AddResource(lr)
	lr.registerDependencies(stack)
	return lr
}

func (lr *ListenerRule) SetStatus(status ListenerRuleStatus) {
	lr.Status = &status
}

func (lr *ListenerRule) registerDependencies(stack core.Manager) {
	for _, dep := range lr.Spec.ListenerID.Dependencies() {
		_ = stack.AddDependency(dep, lr)
	}
}

type ListenerRuleStatus struct {
	RuleID string `json:"ruleID"`
}

type ListenerRuleSpec struct {
	ListenerID core.StringToken `json:"listenerID"`
	ALBListenerRuleSpec
}

type ResAndSDKListenerRulePair struct {
	ResLR *ListenerRule
	SdkLR *albsdk.Rule
}

type ResAndSDKListenerRulePairArray []ResAndSDKListenerRulePair

func (r ResAndSDKListenerRulePairArray) Len() int {
	return len(r)
}

func (r ResAndSDKListenerRulePairArray) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r ResAndSDKListenerRulePairArray) Less(i, j int) bool {
	return r[i].SdkLR.Priority < r[j].SdkLR.Priority
}
