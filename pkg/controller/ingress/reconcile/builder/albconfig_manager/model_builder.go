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
	Build(ctx context.Context, gateway *v1.AlbConfig, ingGroup *Group) (core.Manager, *alb.AlbLoadBalancer, error)
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

func (b defaultAlbConfigManagerBuilder) Build(ctx context.Context, albconfig *v1.AlbConfig, ingGroup *Group) (core.Manager, *alb.AlbLoadBalancer, error) {
	stack := core.NewDefaultManager(core.StackID(ingGroup.ID))

	vpcID, err := b.cloud.VpcID()
	if err != nil {
		return nil, nil, err
	}

	task := &defaultModelBuildTask{
		stack:      stack,
		albconfig:  albconfig,
		ingGroup:   ingGroup,
		kubeClient: b.kubeClient,

		clusterID: b.cloud.ClusterID(),
		vpcID:     vpcID,

		sgpByResID:      make(map[string]*alb.ServerGroup),
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
		return nil, nil, err
	}

	return task.stack, task.loadBalancer, nil
}

type defaultModelBuildTask struct {
	stack        core.Manager
	loadBalancer *alb.AlbLoadBalancer
	albconfig    *v1.AlbConfig
	ingGroup     *Group
	kubeClient   client.Client

	clusterID string
	vpcID     string

	sgpByResID map[string]*alb.ServerGroup

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

type portProtocol struct {
	port     int32
	protocol Protocol
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

func (t *defaultModelBuildTask) run(ctx context.Context) error {
	if !t.albconfig.DeletionTimestamp.IsZero() {
		return nil
	}

	lb, err := t.buildAlbLoadBalancer(ctx, t.albconfig)
	if err != nil {
		return err
	}

	var lss = make(map[int32]*alb.Listener)
	for _, ls := range t.albconfig.Spec.Listeners {
		modelLs, err := t.buildListener(ctx, lb.LoadBalancerID(), ls)
		if err != nil {
			return err
		}
		lss[int32(ls.Port.IntValue())] = modelLs
		err = t.buildAcl(ctx, modelLs, ls, lb)
		if err != nil {
			return err
		}

	}

	ingListByPort := make(map[portProtocol][]networking.Ingress)

	for _, member := range t.ingGroup.Members {
		listenPorts, err := ComputeIngressListenPorts(member)
		if err != nil {
			return err
		}
		for k, v := range listenPorts {
			pp := portProtocol{
				port:     k,
				protocol: v,
			}
			ingListByPort[pp] = append(ingListByPort[pp], *member)
		}
	}
	for pp, ingList := range ingListByPort {
		ls, ok := lss[pp.port]
		if !ok {
			continue
		}
		if pp.protocol == ProtocolHTTPS {
			if len(ls.Spec.Certificates) == 0 {
				var certs []alb.Certificate
				var certsAuto []alb.Certificate
				var certsSecr []alb.Certificate
				for _, ing := range ingList {
					var missHosts []string
					var secretHosts []string
					for _, tls := range ing.Spec.TLS {
						if tls.SecretName != "" {
							cert, err := t.buildSecretCertificate(ctx, ing, tls.SecretName, t.clusterID)
							if err != nil {
								klog.Errorf("build SecretCertificate by secret failed, error: %s", err.Error())
								return err
							}
							certsSecr = append(certsSecr, cert)
							secretHosts = append(secretHosts, tls.Hosts...)
						} else {
							missHosts = append(missHosts, tls.Hosts...)
						}
					}
					for _, rule := range ing.Spec.Rules {
						if rule.Host != "" {
							missHosts = append(missHosts, rule.Host)
						}
					}
					hosts := getStringsDiff(missHosts, secretHosts)
					if len(hosts) > 0 {
						certIds, err := t.computeHostsInferredTLSCertIDs(ctx, hosts)
						if err != nil {
							klog.Errorf("computeIngressInferredTLSCertARNs error: %s", err.Error())
							return err
						}
						certIds = removeDuplicateElement(certIds)
						sort.Strings(certIds)
						for _, cid := range certIds {
							cert := &alb.FixedCertificate{
								IsDefault:     false,
								CertificateId: cid,
							}
							certsAuto = append(certsAuto, cert)
						}
					}
				}
				certs = append(certs, certsAuto...)
				certs = append(certs, certsSecr...)
				if len(certs) > 0 {
					certs[0].SetDefault()
				}
				lss[pp.port].Spec.ListenerProtocol = string(ProtocolHTTPS)
				lss[pp.port].Spec.Certificates = certs
			}
		}
		if err := t.buildListenerRules(ctx, ls.ListenerID(), pp.port, ingList); err != nil {
			return err
		}
	}

	for _, ls := range lss {
		if ls.Spec.ListenerProtocol == string(ProtocolHTTPS) {
			var isDefaultCertExist bool
			for _, c := range ls.Spec.Certificates {
				if c.GetIsDefault() {
					isDefaultCertExist = true
					break
				}
			}
			if len(ls.Spec.Certificates) > 0 && !isDefaultCertExist {
				return fmt.Errorf("https listener: %d must provider one default cert", ls.Spec.ListenerPort)
			}
		}
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

func ComputeIngressListenPorts(ing *networking.Ingress) (map[int32]Protocol, error) {
	rawListenPorts := ""
	portAndProtocols := make(map[int32]Protocol)
	// http transfer to https
	if v := annotations.GetStringAnnotationMutil(annotations.NginxSslRedirect, annotations.AlbSslRedirect, ing); v == "true" {
		portAndProtocols[80] = ProtocolHTTP
	}
	rawListenPorts, err := annotations.GetStringAnnotation(annotations.ListenPorts, ing)
	if err != nil {
		for _, tls := range ing.Spec.TLS {
			for _, host := range tls.Hosts {
				if host != "" {
					portAndProtocols[443] = ProtocolHTTPS
					return portAndProtocols, nil
				}
			}
		}
		return map[int32]Protocol{80: ProtocolHTTP}, nil
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
				portAndProtocols[port] = util.ListenerProtocolHTTP
			case string(ProtocolHTTPS):
				portAndProtocols[port] = util.ListenerProtocolHTTPS
			default:
				return nil, errors.Errorf("listen protocol must be within [%v, %v]: %v", ProtocolHTTP, ProtocolHTTPS, protocol)
			}
		}
	}
	return portAndProtocols, nil
}

type Protocol string

const (
	ProtocolHTTP  Protocol = util.ListenerProtocolHTTP
	ProtocolHTTPS Protocol = util.ListenerProtocolHTTPS
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
