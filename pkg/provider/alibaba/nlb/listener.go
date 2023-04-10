package nlb

import (
	"context"
	"fmt"

	nlb "github.com/alibabacloud-go/nlb-20220430/client"
	"github.com/alibabacloud-go/tea/tea"
	nlbmodel "k8s.io/alibaba-load-balancer-controller/pkg/model/nlb"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/util"
	"k8s.io/klog/v2"
)

func (p *NLBProvider) ListNLBListeners(ctx context.Context, lbId string) ([]*nlbmodel.ListenerAttribute, error) {
	var respListeners []*nlb.ListListenersResponseBodyListeners
	nextToken := ""
	for {
		req := &nlb.ListListenersRequest{}
		req.LoadBalancerIds = []*string{tea.String(lbId)}
		req.MaxResults = tea.Int32(100)
		req.NextToken = tea.String(nextToken)

		resp, err := p.auth.NLB.ListListeners(req)
		if err != nil {
			return nil, util.SDKError("ListListeners", err)
		}
		if resp == nil || resp.Body == nil {
			return nil, fmt.Errorf("OpenAPI ListNLBListeners resp is nil")
		}
		respListeners = append(respListeners, resp.Body.Listeners...)

		nextToken = tea.StringValue(resp.Body.NextToken)
		if nextToken == "" {
			break
		}
	}

	var listeners []*nlbmodel.ListenerAttribute
	for _, lis := range respListeners {
		n := &nlbmodel.ListenerAttribute{
			ListenerId:          tea.StringValue(lis.ListenerId),
			ListenerDescription: tea.StringValue(lis.ListenerDescription),
			ListenerProtocol:    tea.StringValue(lis.ListenerProtocol),
			ListenerPort:        tea.Int32Value(lis.ListenerPort),
			ServerGroupId:       tea.StringValue(lis.ServerGroupId),
			ListenerStatus:      nlbmodel.ListenerStatus(tea.StringValue(lis.ListenerStatus)),
		}
		if lis.IdleTimeout != nil {
			n.IdleTimeout = tea.Int32Value(lis.IdleTimeout)
		}
		if lis.SecurityPolicyId != nil {
			n.SecurityPolicyId = tea.StringValue(lis.SecurityPolicyId)
		}
		for _, c := range lis.CertificateIds {
			if c != nil {
				n.CertificateIds = append(n.CertificateIds, tea.StringValue(c))
			}
		}
		for _, c := range lis.CaCertificateIds {
			if c != nil {
				n.CaCertificateIds = append(n.CaCertificateIds, tea.StringValue(c))
			}
		}
		n.CaEnabled = lis.CaEnabled
		n.Cps = lis.Cps
		n.ProxyProtocolEnabled = lis.ProxyProtocolEnabled
		nameKey, err := nlbmodel.LoadNLBListenerNamedKey(n.ListenerDescription)
		if err != nil {
			n.IsUserManaged = true
			klog.Warningf("listener description [%s], not expected format. skip user managed port",
				tea.StringValue(lis.ListenerDescription))
		}
		n.NamedKey = nameKey

		listeners = append(listeners, n)
	}
	return listeners, nil
}

func (p *NLBProvider) CreateNLBListener(ctx context.Context, lbId string, lis *nlbmodel.ListenerAttribute) error {
	req := &nlb.CreateListenerRequest{}
	req.LoadBalancerId = tea.String(lbId)
	req.ListenerProtocol = tea.String(lis.ListenerProtocol)
	req.ListenerPort = tea.Int32(lis.ListenerPort)
	req.ListenerDescription = tea.String(lis.ListenerDescription)
	req.ServerGroupId = tea.String(lis.ServerGroupId)
	req.Cps = lis.Cps
	req.ProxyProtocolEnabled = lis.ProxyProtocolEnabled
	if lis.IdleTimeout != 0 {
		req.IdleTimeout = tea.Int32(lis.IdleTimeout)
	}
	if lis.SecurityPolicyId != "" {
		req.SecurityPolicyId = tea.String(lis.SecurityPolicyId)
	}
	for _, cert := range lis.CertificateIds {
		req.CertificateIds = append(req.CertificateIds, tea.String(cert))
	}
	for _, cert := range lis.CaCertificateIds {
		req.CaCertificateIds = append(req.CaCertificateIds, tea.String(cert))
	}
	req.CaEnabled = lis.CaEnabled

	_, err := p.auth.NLB.CreateListener(req)
	return util.SDKError("CreateListener", err)
}

func (p *NLBProvider) UpdateNLBListener(ctx context.Context, lis *nlbmodel.ListenerAttribute) error {
	req := &nlb.UpdateListenerAttributeRequest{}
	req.ListenerId = tea.String(lis.ListenerId)
	req.ListenerDescription = tea.String(lis.ListenerDescription)
	req.ServerGroupId = tea.String(lis.ServerGroupId)
	req.Cps = lis.Cps
	req.ProxyProtocolEnabled = lis.ProxyProtocolEnabled
	if lis.IdleTimeout != 0 {
		req.IdleTimeout = tea.Int32(lis.IdleTimeout)
	}
	if lis.SecurityPolicyId != "" {
		req.SecurityPolicyId = tea.String(lis.SecurityPolicyId)
	}
	for _, cert := range lis.CertificateIds {
		req.CertificateIds = append(req.CertificateIds, tea.String(cert))
	}
	for _, cert := range lis.CaCertificateIds {
		req.CaCertificateIds = append(req.CaCertificateIds, tea.String(cert))
	}
	req.CaEnabled = lis.CaEnabled

	_, err := p.auth.NLB.UpdateListenerAttribute(req)
	return util.SDKError("UpdateListenerAttribute", err)
}

func (p *NLBProvider) DeleteNLBListener(ctx context.Context, listenerId string) error {
	req := &nlb.DeleteListenerRequest{}
	req.ListenerId = tea.String(listenerId)

	resp, err := p.auth.NLB.DeleteListener(req)
	if err != nil {
		return util.SDKError("DeleteNLBListener", err)
	}
	if resp == nil || resp.Body == nil {
		return fmt.Errorf("OpenAPI DeleteNLBListener resp is nil")
	}
	return p.waitJobFinish("DeleteListener", tea.StringValue(resp.Body.JobId))
}

func (p *NLBProvider) StartNLBListener(ctx context.Context, listenerId string) error {
	req := &nlb.StartListenerRequest{}
	req.ListenerId = tea.String(listenerId)

	_, err := p.auth.NLB.StartListener(req)
	return util.SDKError("StartListener", err)
}
