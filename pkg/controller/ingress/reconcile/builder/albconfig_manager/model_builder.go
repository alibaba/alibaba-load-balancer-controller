package albconfigmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	v1 "k8s.io/alibaba-load-balancer-controller/pkg/apis/alibabacloud/v1"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/annotations"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb/core"
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type Builder interface {
	Build(ctx context.Context, gateway *v1.AlbConfig, ingGroup *Group) (core.Manager, *alb.AlbLoadBalancer, map[*networking.Ingress]error, error)
}

var _ Builder = &defaultAlbConfigManagerBuilder{}

type defaultAlbConfigManagerBuilder struct {
	kubeClient client.Client
	cloud      prvd.Provider
	logger     logr.Logger
}

func NewDefaultAlbConfigManagerBuilder(kubeClient client.Client, cloud prvd.Provider, logger logr.Logger) *defaultAlbConfigManagerBuilder {
	return &defaultAlbConfigManagerBuilder{
		kubeClient: kubeClient,
		cloud:      cloud,
		logger:     logger,
	}
}

func (b defaultAlbConfigManagerBuilder) Build(ctx context.Context, albconfig *v1.AlbConfig, ingGroup *Group) (core.Manager, *alb.AlbLoadBalancer, map[*networking.Ingress]error, error) {
	stack := core.NewDefaultManager(core.StackID(ingGroup.ID))
	errResultWithIngress := make(map[*networking.Ingress]error)
	vpcID, err := b.cloud.VpcID()
	if err != nil {
		return nil, nil, errResultWithIngress, err
	}

	task := &defaultModelBuildTask{
		stack:                stack,
		albconfig:            albconfig,
		ingGroup:             ingGroup,
		kubeClient:           b.kubeClient,
		errResultWithIngress: errResultWithIngress,

		clusterID: b.cloud.ClusterID(),
		vpcID:     vpcID,

		sgpByResID:      make(map[string]*alb.ServerGroup),
		scByResID:       make(map[string]*alb.SecretCertificate),
		backendServices: make(map[types.NamespacedName]*corev1.Service),

		annotationParser: annotations.NewSuffixAnnotationParser(annotations.DefaultAnnotationsPrefix),
		certDiscovery:    NewCASCertDiscovery(b.cloud, b.logger),
		vSwitchResolver:  NewDefaultVSwitchResolver(b.cloud, vpcID, b.logger),

		defaultServerGroupScheduler:     util.DefaultServerGroupScheduler,
		defaultServerGroupProtocol:      util.DefaultServerGroupProtocol,
		defaultServerGroupType:          util.DefaultServerGroupType,
		defaultListenerProtocol:         util.DefaultListenerProtocol,
		defaultListenerPort:             util.DefaultListenerPort,
		defaultListenerIdleTimeout:      util.DefaultListenerIdleTimeout,
		defaultListenerRequestTimeout:   util.DefaultListenerRequestTimeout,
		defaultListenerGzipEnabled:      util.DefaultListenerGzipEnabled,
		defaultListenerHttp2Enabled:     util.DefaultListenerHttp2Enabled,
		defaultListenerSecurityPolicyId: util.DefaultListenerSecurityPolicyId,
	}
	if err := task.run(ctx); err != nil {
		return nil, nil, errResultWithIngress, err
	}

	return task.stack, task.loadBalancer, errResultWithIngress, nil
}

type defaultModelBuildTask struct {
	stack                core.Manager
	loadBalancer         *alb.AlbLoadBalancer
	albconfig            *v1.AlbConfig
	ingGroup             *Group
	kubeClient           client.Client
	errResultWithIngress map[*networking.Ingress]error

	clusterID string
	vpcID     string

	sgpByResID map[string]*alb.ServerGroup
	scByResID  map[string]*alb.SecretCertificate

	annotationParser annotations.Parser
	certDiscovery    CertDiscovery
	vSwitchResolver  VSwitchResolver

	backendServices map[types.NamespacedName]*corev1.Service

	defaultServerGroupScheduler string
	defaultServerGroupProtocol  string
	defaultServerGroupType      string

	defaultListenerPort             int
	defaultListenerProtocol         string
	defaultListenerIdleTimeout      int
	defaultListenerRequestTimeout   int
	defaultListenerGzipEnabled      bool
	defaultListenerHttp2Enabled     bool
	defaultListenerSecurityPolicyId string
}

type PortProtocol struct {
	Port     int32
	Protocol Protocol
}

var (
	fakeDefaultServiceName = "fake-svc"
)

func (t *defaultModelBuildTask) buildLsDefaultAction(ctx context.Context, lsPort int) (alb.Action, error) {
	svcName := fakeDefaultServiceName
	ing := new(networking.Ingress)
	ing.Namespace = t.albconfig.Namespace
	if ing.Namespace == "" {
		ing.Namespace = ALBConfigNamespace
	}
	ing.Name = t.albconfig.Name + util.DefaultListenerFlag + strconv.Itoa(lsPort)
	action := buildActionViaServiceAndServicePort(ctx, svcName, lsPort, 0)
	actions, err := t.buildAction(ctx, *ing, action)
	if err != nil {
		t.errResultWithIngress[ing] = err
		return alb.Action{}, err
	}

	return actions, nil
}

func removeDuplicateElement(elements []string) []string {
	result := make([]string, 0, len(elements))
	temp := map[string]struct{}{}
	for _, element := range elements {
		if _, ok := temp[element]; !ok {
			temp[element] = struct{}{}
			result = append(result, element)
		}
	}
	return result
}

func removeDuplicateSecretCertificate(elements []*alb.SecretCertificate) []*alb.SecretCertificate {
	result := make([]*alb.SecretCertificate, 0, len(elements))
	temp := map[string]struct{}{}
	for _, element := range elements {
		if _, ok := temp[element.Spec.CertName]; !ok {
			temp[element.Spec.CertName] = struct{}{}
			result = append(result, element)
		}
	}
	return result
}

func (t *defaultModelBuildTask) run(ctx context.Context) error {
	if !t.albconfig.DeletionTimestamp.IsZero() {
		return nil
	}
	lb, err := t.buildAlbLoadBalancer(ctx, t.albconfig)
	if err != nil {
		return err
	}

	var lss = make(map[PortProtocol]*alb.Listener)
	for _, ls := range t.albconfig.Spec.Listeners {
		modelLs, err := t.buildListener(ctx, lb.LoadBalancerID(), ls)
		if err != nil {
			return err
		}
		pp := PortProtocol{
			Port:     int32(ls.Port.IntValue()),
			Protocol: Protocol(ls.Protocol),
		}
		lss[pp] = modelLs
		err = t.buildAcl(ctx, modelLs, ls, lb)
		if err != nil {
			return err
		}

	}

	ingListByPort := make(map[PortProtocol][]networking.Ingress)

	for _, member := range t.ingGroup.Members {
		listenPorts, err := ComputeIngressListenPorts(member)
		if err != nil {
			t.errResultWithIngress[member] = err
			return err
		}
		for _, pp := range listenPorts {
			ingListByPort[pp] = append(ingListByPort[pp], *member)
		}
	}
	for pp, ingList := range ingListByPort {
		ls, ok := lss[pp]
		if !ok {
			continue
		}
		if err := t.buildListenerRules(ctx, ls.ListenerID(), pp.Port, pp.Protocol, ingList); err != nil {
			return err
		}
		if pp.Protocol != ProtocolHTTPS && pp.Protocol != ProtocolQUIC {
			continue
		}
		var certsSecCert []*alb.SecretCertificate
		var missHosts []string
		var secretHosts []string
		for _, ing := range ingList {
			for _, tls := range ing.Spec.TLS {
				if tls.SecretName != "" {
					cert, err := t.buildSecretCertificate(ctx, ing, tls.SecretName, t.clusterID, pp)
					if err != nil {
						t.errResultWithIngress[&ing] = err
						klog.Errorf("build SecretCertificate by secret failed, error: %s", err.Error())
						return err
					}
					certsSecCert = append(certsSecCert, cert)
					secretHosts = append(secretHosts, tls.Hosts...)
				} else {
					missHosts = append(missHosts, tls.Hosts...)
				}
			}
			for _, rule := range ing.Spec.Rules {
				// no discovery certificate for rule.Host if not config in tls.host
				if rule.Host != "" && withTlsRule(pp.Protocol, rule.Host, ing) {
					missHosts = append(missHosts, rule.Host)
				}
			}
		}
		var certsSecr []alb.Certificate
		certsSecCert = removeDuplicateSecretCertificate(certsSecCert)
		for _, sc := range certsSecCert {
			certsSecr = append(certsSecr, sc)
		}

		var certsFixed []alb.Certificate
		hosts := getStringsDiff(missHosts, secretHosts)
		if len(ls.Spec.Certificates) != 0 || len(hosts) == 0 {
			certsFixed = ls.Spec.Certificates
		} else {
			certIds, err := t.computeHostsInferredTLSCertIDs(ctx, hosts)
			if err != nil {
				klog.Errorf("computeIngressInferredTLSCertARNs error: %s", err.Error())
				return err
			}
			certsID := removeDuplicateElement(certIds)
			sort.Strings(certsID)
			for _, cid := range certsID {
				cert := &alb.FixedCertificate{
					IsDefault:     false,
					CertificateId: cid,
				}
				certsFixed = append(certsFixed, cert)
			}
		}
		var certs []alb.Certificate
		certs = append(certs, certsFixed...)
		certs = append(certs, certsSecr...)
		if len(certs) > 0 {
			certs[0].SetDefault()
		}
		lss[pp].Spec.ListenerProtocol = string(pp.Protocol)
		lss[pp].Spec.Certificates = certs
	}

	return nil
}

func (t *defaultModelBuildTask) computeHostsInferredTLSCertIDs(ctx context.Context, hosts []string) ([]string, error) {
	dHosts := sets.NewString()
	for _, h := range hosts {
		dHosts.Insert(h)
	}
	return t.certDiscovery.Discover(ctx, dHosts.List())
}

func ComputeIngressListenPorts(ing *networking.Ingress) ([]PortProtocol, error) {
	rawListenPorts := ""
	portAndProtocols := []PortProtocol{}

	if v := annotations.GetStringAnnotationMutil(annotations.NginxSslRedirect, annotations.AlbSslRedirect, ing); v == "true" {
		portAndProtocols = append(portAndProtocols, getPPByPortProtol(80, ProtocolHTTP))
	}
	rawListenPorts, err := annotations.GetStringAnnotation(annotations.ListenPorts, ing)
	if err != nil {
		ls443 := false
		for _, tls := range ing.Spec.TLS {
			for _, host := range tls.Hosts {
				if host != "" {
					ls443 = true
					break
				}
			}
			if ls443 {
				break
			}
		}
		ls80 := false
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" && !withTlsRule(ProtocolHTTP, rule.Host, *ing) {
				ls80 = true
				break
			}
		}

		if ls443 && ls80 {
			portAndProtocols = append(portAndProtocols, getPPByPortProtol(80, ProtocolHTTP), getPPByPortProtol(443, ProtocolHTTPS))
		} else if ls443 {
			portAndProtocols = append(portAndProtocols, getPPByPortProtol(443, ProtocolHTTPS))
		} else {
			portAndProtocols = append(portAndProtocols, getPPByPortProtol(80, ProtocolHTTP))
		}
		return RemoveDuplicatePPElement(portAndProtocols), nil
	}

	var entries []map[string]int32
	if err := json.Unmarshal([]byte(rawListenPorts), &entries); err != nil {
		return nil, errors.Wrapf(err, "failed to parse listen-ports configuration: `%s`", rawListenPorts)
	}
	if len(entries) == 0 {
		return nil, errors.Errorf("empty listen-ports configuration: `%s`", rawListenPorts)
	}

	for _, entry := range entries {
		for protocol, port := range entry {
			if port < 1 || port > 65535 {
				return nil, errors.Errorf("listen port must be within [1, 65535]: %v", port)
			}
			switch protocol {
			case string(ProtocolHTTP):
				portAndProtocols = append(portAndProtocols, getPPByPortProtol(port, ProtocolHTTP))
			case string(ProtocolHTTPS):
				portAndProtocols = append(portAndProtocols, getPPByPortProtol(port, ProtocolHTTPS))
			case string(ProtocolQUIC):
				portAndProtocols = append(portAndProtocols, getPPByPortProtol(port, ProtocolQUIC))
			default:
				return nil, errors.Errorf("listen protocol must be within [%v, %v]: %v", ProtocolHTTP, ProtocolHTTPS, protocol)
			}
		}
	}
	return RemoveDuplicatePPElement(portAndProtocols), nil
}

func getPPByPortProtol(port int32, protocol Protocol) PortProtocol {
	return PortProtocol{
		Port:     port,
		Protocol: protocol,
	}
}

func RemoveDuplicatePPElement(elements []PortProtocol) []PortProtocol {
	result := make([]PortProtocol, 0, len(elements))
	temp := map[string]struct{}{}
	for _, element := range elements {
		key := fmt.Sprint(element.Port, element.Protocol)
		if _, ok := temp[key]; !ok {
			temp[key] = struct{}{}
			result = append(result, element)
		}
	}
	return result
}

type Protocol string

const (
	ProtocolHTTP  Protocol = util.ListenerProtocolHTTP
	ProtocolHTTPS Protocol = util.ListenerProtocolHTTPS
	ProtocolQUIC  Protocol = util.ListenerProtocolQUIC
)

func getStringsDiff(a, b []string) []string {
	mapB := make(map[string]bool)
	for _, s := range b {
		mapB[s] = true
	}
	aDiffB := make([]string, 0)
	for _, s := range a {
		if !mapB[s] {
			aDiffB = append(aDiffB, s)
		}
	}
	return aDiffB
}
