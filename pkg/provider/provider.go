package prvd

import (
	"context"
	"time"

	"k8s.io/alibaba-load-balancer-controller/pkg/model/tag"

	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress/reconcile/tracking"

	albmodel "k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
	nlbmodel "k8s.io/alibaba-load-balancer-controller/pkg/model/nlb"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
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
	ILoadBalancer
	IALB
	INLB
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
	ListInstances(ctx context.Context, ids []string) (map[string]*NodeAttribute, error)
	GetInstancesByIP(ctx context.Context, ips []string) (*NodeAttribute, error)
	// DescribeNetworkInterfaces query one or more elastic network interfaces (ENIs)
	GetInstanceByIp(ip, region, vpc string) ([]ecs.Instance, error)
	DescribeNetworkInterfaces(vpcId string, ips []string, ipVersionType model.AddressIPVersionType) (map[string]string, error)
}

type IVPC interface {
	DescribeVSwitches(ctx context.Context, vpcID string) ([]vpc.VSwitch, error)
}

type ILoadBalancer interface {
	// LoadBalancer
	FindLoadBalancer(ctx context.Context, mdl *model.LoadBalancer) error
	CreateLoadBalancer(ctx context.Context, mdl *model.LoadBalancer) error
	DescribeLoadBalancer(ctx context.Context, mdl *model.LoadBalancer) error
	DeleteLoadBalancer(ctx context.Context, mdl *model.LoadBalancer) error
	ModifyLoadBalancerInstanceSpec(ctx context.Context, lbId string, spec string) error
	ModifyLoadBalancerInstanceChargeType(ctx context.Context, lbId string, instanceChargeType string, spec string) error
	SetLoadBalancerDeleteProtection(ctx context.Context, lbId string, flag string) error
	SetLoadBalancerName(ctx context.Context, lbId string, name string) error
	ModifyLoadBalancerInternetSpec(ctx context.Context, lbId string, chargeType string, bandwidth int) error
	SetLoadBalancerModificationProtection(ctx context.Context, lbId string, flag string) error

	// Listener
	DescribeLoadBalancerListeners(ctx context.Context, lbId string) ([]model.ListenerAttribute, error)
	StartLoadBalancerListener(ctx context.Context, lbId string, port int) error
	StopLoadBalancerListener(ctx context.Context, lbId string, port int) error
	DeleteLoadBalancerListener(ctx context.Context, lbId string, port int) error
	CreateLoadBalancerTCPListener(ctx context.Context, lbId string, listener model.ListenerAttribute) error
	SetLoadBalancerTCPListenerAttribute(ctx context.Context, lbId string, listener model.ListenerAttribute) error
	CreateLoadBalancerUDPListener(ctx context.Context, lbId string, listener model.ListenerAttribute) error
	SetLoadBalancerUDPListenerAttribute(ctx context.Context, lbId string, listener model.ListenerAttribute) error
	CreateLoadBalancerHTTPListener(ctx context.Context, lbId string, listener model.ListenerAttribute) error
	SetLoadBalancerHTTPListenerAttribute(ctx context.Context, lbId string, listener model.ListenerAttribute) error
	CreateLoadBalancerHTTPSListener(ctx context.Context, lbId string, listener model.ListenerAttribute) error
	SetLoadBalancerHTTPSListenerAttribute(ctx context.Context, lbId string, listener model.ListenerAttribute) error

	// VServerGroup
	DescribeVServerGroups(ctx context.Context, lbId string) ([]model.VServerGroup, error)
	CreateVServerGroup(ctx context.Context, vg *model.VServerGroup, lbId string) error
	DescribeVServerGroupAttribute(ctx context.Context, vGroupId string) (model.VServerGroup, error)
	DeleteVServerGroup(ctx context.Context, vGroupId string) error
	AddVServerGroupBackendServers(ctx context.Context, vGroupId string, backends string) error
	RemoveVServerGroupBackendServers(ctx context.Context, vGroupId string, backends string) error
	SetVServerGroupAttribute(ctx context.Context, vGroupId string, backends string) error
	ModifyVServerGroupBackendServers(ctx context.Context, vGroupId string, old string, new string) error

	// Tag
	TagCLBResource(ctx context.Context, resourceId string, tags []tag.Tag) error
	ListCLBTagResources(ctx context.Context, lbId string) ([]tag.Tag, error)
}

// type IPrivateZone interface {
// 	ListPVTZ(ctx context.Context) ([]*model.PvtzEndpoint, error)
// 	SearchPVTZ(ctx context.Context, ep *model.PvtzEndpoint, exact bool) ([]*model.PvtzEndpoint, error)
// 	UpdatePVTZ(ctx context.Context, ep *model.PvtzEndpoint) error
// 	DeletePVTZ(ctx context.Context, ep *model.PvtzEndpoint) error
// }

type ISLS interface {
	AnalyzeProductLog(request *sls.AnalyzeProductLogRequest) (response *sls.AnalyzeProductLogResponse, err error)
	SLSDoAction(request requests.AcsRequest, response responses.AcsResponse) (err error)
}

type ICAS interface {
	DescribeSSLCertificateList(ctx context.Context) ([]model.CertificateInfo, error)
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
	UpdateALB(ctx context.Context, resLB *albmodel.AlbLoadBalancer, sdkLB alb.LoadBalancer, trackingProvider tracking.TrackingProvider) (albmodel.LoadBalancerStatus, error)
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
	SelectALBServerGroupsByID(ctx context.Context, serverGroupID string) (albmodel.ServerGroupWithTags, error)

	// ALB Tags
	ListALBServerGroupsWithTags(ctx context.Context, tagFilters map[string]string) ([]albmodel.ServerGroupWithTags, error)
	ListALBsWithTags(ctx context.Context, tagFilters map[string]string) ([]albmodel.AlbLoadBalancerWithTags, error)

	DoAction(request requests.AcsRequest, response responses.AcsResponse) (err error)

	// ACL support

	CreateAcl(ctx context.Context, resAcl *albmodel.Acl) (albmodel.AclStatus, error)
	UpdateAcl(ctx context.Context, listenerID string, resAndSDKAclPair albmodel.ResAndSDKAclPair) (albmodel.AclStatus, error)
	DeleteAcl(ctx context.Context, listenerID, sdkAclID string) error
	ListAcl(ctx context.Context, listener *albmodel.Listener, aclIds []string) ([]alb.Acl, error)
	ListAclEntriesByID(traceID interface{}, sdkAclID string) ([]alb.AclEntry, error)
	AssociateAclWithListener(ctx context.Context, traceID interface{}, resAcl *albmodel.Acl, aclIds []string) error
	DisassociateAclWithListener(traceID interface{}, listenerID string, aclIds []string) error
}

type INLB interface {
	//Tag
	TagNLBResource(ctx context.Context, resourceId string, resourceType nlbmodel.TagResourceType, tags []tag.Tag) error
	ListNLBTagResources(ctx context.Context, lbId string) ([]tag.Tag, error)
	// NetworkLoadBalancer
	FindNLB(ctx context.Context, mdl *nlbmodel.NetworkLoadBalancer) error
	DescribeNLB(ctx context.Context, mdl *nlbmodel.NetworkLoadBalancer) error
	CreateNLB(ctx context.Context, mdl *nlbmodel.NetworkLoadBalancer) error
	DeleteNLB(ctx context.Context, mdl *nlbmodel.NetworkLoadBalancer) error
	UpdateNLB(ctx context.Context, mdl *nlbmodel.NetworkLoadBalancer) error
	UpdateNLBAddressType(ctx context.Context, mdl *nlbmodel.NetworkLoadBalancer) error
	UpdateNLBZones(ctx context.Context, mdl *nlbmodel.NetworkLoadBalancer) error

	// ServerGroup
	ListNLBServerGroups(ctx context.Context, tags []tag.Tag) ([]*nlbmodel.ServerGroup, error)
	CreateNLBServerGroup(ctx context.Context, sg *nlbmodel.ServerGroup) error
	DeleteNLBServerGroup(ctx context.Context, sgId string) error
	UpdateNLBServerGroup(ctx context.Context, sg *nlbmodel.ServerGroup) error
	AddNLBServers(ctx context.Context, sgId string, backends []nlbmodel.ServerGroupServer) error
	RemoveNLBServers(ctx context.Context, sgId string, backends []nlbmodel.ServerGroupServer) error
	UpdateNLBServers(ctx context.Context, sgId string, backends []nlbmodel.ServerGroupServer) error

	// Listener
	ListNLBListeners(ctx context.Context, lbId string) ([]*nlbmodel.ListenerAttribute, error)
	CreateNLBListener(ctx context.Context, lbId string, lis *nlbmodel.ListenerAttribute) error
	UpdateNLBListener(ctx context.Context, lis *nlbmodel.ListenerAttribute) error
	DeleteNLBListener(ctx context.Context, listenerId string) error
	StartNLBListener(ctx context.Context, listenerId string) error
}
