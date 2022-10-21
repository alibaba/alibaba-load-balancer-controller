package prvd

import (
	"context"
	"time"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/tracking"

	albmodel "k8s.io/alibaba-load-balancer-controller/pkg/model/alb"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cas"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sls"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"k8s.io/alibaba-load-balancer-controller/pkg/model"
	v1 "k8s.io/api/core/v1"
)

type Provider interface {
	IMetaData
	IInstance
	IVPC
	IALB
	ISLS
	ICAS
}

type RoleAuth struct {
	AccessKeyId     string
	AccessKeySecret string
	Expiration      time.Time
	SecurityToken   string
	LastUpdated     time.Time
	Code            string
}

// IMetaData metadata interface
type IMetaData interface {
	// values from metadata server
	HostName() (string, error)
	ImageID() (string, error)
	InstanceID() (string, error)
	Mac() (string, error)
	NetworkType() (string, error)
	OwnerAccountID() (string, error)
	PrivateIPv4() (string, error)
	Region() (string, error)
	SerialNumber() (string, error)
	SourceAddress() (string, error)
	VpcCIDRBlock() (string, error)
	VpcID() (string, error)
	VswitchCIDRBlock() (string, error)
	Zone() (string, error)
	NTPConfigServers() ([]string, error)
	RoleName() (string, error)
	RamRoleToken(role string) (RoleAuth, error)
	VswitchID() (string, error)
	// values from cloud config file
	ClusterID() string
}

// NodeAttribute node attribute from cloud instance
type NodeAttribute struct {
	InstanceID   string
	Addresses    []v1.NodeAddress
	InstanceType string
	Zone         string
	Region       string
}

type IInstance interface {
	GetInstanceByIp(ip, region, vpc string) ([]ecs.Instance, error)
	DescribeNetworkInterfaces(vpcId string, ips []string, ipVersionType model.AddressIPVersionType) (map[string]string, error)
}

type IVPC interface {
	DescribeVSwitches(ctx context.Context, vpcID string) ([]vpc.VSwitch, error)
}

type ISLS interface {
	AnalyzeProductLog(request *sls.AnalyzeProductLogRequest) (response *sls.AnalyzeProductLogResponse, err error)
	SLSDoAction(request requests.AcsRequest, response responses.AcsResponse) (err error)
}

type ICAS interface {
	DescribeSSLCertificateList(ctx context.Context) ([]cas.CertificateInfo, error)
	DescribeSSLCertificatePublicKeyDetail(ctx context.Context, request *cas.DescribeSSLCertificatePublicKeyDetailRequest) (*cas.DescribeSSLCertificatePublicKeyDetailResponse, error)
	CreateSSLCertificateWithName(ctx context.Context, certName, certificate, privateKey string) (string, error)
	DeleteSSLCertificate(ctx context.Context, certId string) error
}

type IALB interface {
	DescribeALBZones(request *alb.DescribeZonesRequest) (response *alb.DescribeZonesResponse, err error)
	TagALBResources(request *alb.TagResourcesRequest) (response *alb.TagResourcesResponse, err error)
	UnTagALBResources(request *alb.UnTagResourcesRequest) (response *alb.UnTagResourcesResponse, err error)
	// ApplicationLoadBalancer
	CreateALB(ctx context.Context, resLB *albmodel.AlbLoadBalancer, trackingProvider tracking.TrackingProvider) (albmodel.LoadBalancerStatus, error)
	ReuseALB(ctx context.Context, resLB *albmodel.AlbLoadBalancer, lbID string, trackingProvider tracking.TrackingProvider) (albmodel.LoadBalancerStatus, error)
	UnReuseALB(ctx context.Context, lbID string, trackingProvider tracking.TrackingProvider) error
	UpdateALB(ctx context.Context, resLB *albmodel.AlbLoadBalancer, sdkLB alb.LoadBalancer) (albmodel.LoadBalancerStatus, error)
	DeleteALB(ctx context.Context, lbID string) error
	// ALB Listener
	CreateALBListener(ctx context.Context, resLS *albmodel.Listener) (albmodel.ListenerStatus, error)
	UpdateALBListener(ctx context.Context, resLS *albmodel.Listener, sdkLB *alb.Listener) (albmodel.ListenerStatus, error)
	DeleteALBListener(ctx context.Context, lsID string) error
	ListALBListeners(ctx context.Context, lbID string) ([]alb.Listener, error)

	// ALB Listener Rule
	CreateALBListenerRule(ctx context.Context, resLR *albmodel.ListenerRule) (albmodel.ListenerRuleStatus, error)
	CreateALBListenerRules(ctx context.Context, resLR []*albmodel.ListenerRule) (map[int]albmodel.ListenerRuleStatus, error)
	UpdateALBListenerRule(ctx context.Context, resLR *albmodel.ListenerRule, sdkLR *alb.Rule) (albmodel.ListenerRuleStatus, error)
	UpdateALBListenerRules(ctx context.Context, matches []albmodel.ResAndSDKListenerRulePair) error
	DeleteALBListenerRule(ctx context.Context, sdkLRId string) error
	DeleteALBListenerRules(ctx context.Context, sdkLRIds []string) error
	ListALBListenerRules(ctx context.Context, lsID string) ([]alb.Rule, error)
	GetALBListenerAttribute(ctx context.Context, lsID string) (*alb.GetListenerAttributeResponse, error)

	// ALB Server
	RegisterALBServers(ctx context.Context, serverGroupID string, resServers []albmodel.BackendItem) error
	DeregisterALBServers(ctx context.Context, serverGroupID string, sdkServers []alb.BackendServer) error
	ReplaceALBServers(ctx context.Context, serverGroupID string, resServers []albmodel.BackendItem, sdkServers []alb.BackendServer) error
	ListALBServers(ctx context.Context, serverGroupID string) ([]alb.BackendServer, error)

	// ALB ServerGroup
	CreateALBServerGroup(ctx context.Context, resSGP *albmodel.ServerGroup, trackingProvider tracking.TrackingProvider) (albmodel.ServerGroupStatus, error)
	UpdateALBServerGroup(ctx context.Context, resSGP *albmodel.ServerGroup, sdkSGP albmodel.ServerGroupWithTags) (albmodel.ServerGroupStatus, error)
	DeleteALBServerGroup(ctx context.Context, serverGroupID string) error

	// ALB Tags
	ListALBServerGroupsWithTags(ctx context.Context, tagFilters map[string]string) ([]albmodel.ServerGroupWithTags, error)
	ListALBsWithTags(ctx context.Context, tagFilters map[string]string) ([]albmodel.AlbLoadBalancerWithTags, error)

	DoAction(request requests.AcsRequest, response responses.AcsResponse) (err error)

	// ACL support

	CreateAcl(ctx context.Context, resAcl *albmodel.Acl) (albmodel.AclStatus, error)
	UpdateAcl(ctx context.Context, listenerID string, resAndSDKAclPair albmodel.ResAndSDKAclPair) (albmodel.AclStatus, error)
	DeleteAcl(ctx context.Context, listenerID, sdkAclID string) error
	ListAcl(ctx context.Context, listener *albmodel.Listener, aclId string) ([]alb.Acl, error)
	ListAclEntriesByID(traceID interface{}, sdkAclID string) ([]alb.AclEntry, error)
}
