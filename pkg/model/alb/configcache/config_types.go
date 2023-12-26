package configcache

type Action struct {
	Order               int                  `json:"Order" xml:"Order"`
	Type                string               `json:"Type" xml:"Type"`
	ForwardConfig       *ForwardActionConfig `json:"forwardConfig,omitempty"`
	FixedResponseConfig *FixedResponseConfig `json:"FixedResponseConfig" xml:"FixedResponseConfig"`
	RedirectConfig      *RedirectConfig      `json:"RedirectConfig" xml:"RedirectConfig"`
	InsertHeaderConfig  *InsertHeaderConfig  `json:"InsertHeaderConfig" xml:"InsertHeaderConfig"`
	RemoveHeaderConfig  *RemoveHeaderConfig  `json:"RemoveHeaderConfig" xml:"RemoveHeaderConfig"`
	RewriteConfig       *RewriteConfig       `json:"RewriteConfig" xml:"RewriteConfig"`
	TrafficMirrorConfig *TrafficMirrorConfig `json:"TrafficMirrorConfig" xml:"TrafficMirrorConfig"`
	TrafficLimitConfig  *TrafficLimitConfig  `json:"TrafficLimitConfig" xml:"TrafficLimitConfig"`
	CorsConfig          *CorsConfig          `json:"CorsConfig" xml:"CorsConfig"`
}

type InsertHeaderConfig struct {
	CoverEnabled bool   `json:"CoverEnabled" xml:"CoverEnabled"`
	Key          string `json:"Key" xml:"Key"`
	Value        string `json:"Value" xml:"Value"`
	ValueType    string `json:"ValueType" xml:"ValueType"`
}
type RemoveHeaderConfig struct {
	Key string `json:"Key" xml:"Key"`
}
type RewriteConfig struct {
	Host  string `json:"Host" xml:"Host"`
	Path  string `json:"Path" xml:"Path"`
	Query string `json:"Query" xml:"Query"`
}
type RedirectConfig struct {
	Host     string `json:"Host" xml:"Host"`
	HttpCode string `json:"HttpCode" xml:"HttpCode"`
	Path     string `json:"Path" xml:"Path"`
	Port     string `json:"Port" xml:"Port"`
	Protocol string `json:"Protocol" xml:"Protocol"`
	Query    string `json:"Query" xml:"Query"`
}

type ForwardActionConfig struct {
	ServerGroupStickySession *ServerGroupStickySession `json:"ServerGroupStickySession" xml:"ServerGroupStickySession"`
	ServerGroups             []ServerGroupTuple        `json:"serverGroups"`
}
type ServerGroupStickySession struct {
	Enabled bool `json:"Enabled" xml:"Enabled"`
	Timeout int  `json:"Timeout" xml:"Timeout"`
}

type ServerGroupTuple struct {
	ServerGroupID string `json:"serverGroupID"`

	ServiceName string `json:"serviceName"`

	ServicePort int `json:"servicePort"`

	Weight int `json:"weight,omitempty"`
}

type TrafficMirrorConfig struct {
	TargetType        string            `json:"TargetType" xml:"TargetType"`
	MirrorGroupConfig MirrorGroupConfig `json:"MirrorGroupConfig" xml:"MirrorGroupConfig"`
}
type TrafficMirrorServerGroupTuple struct {
	ServerGroupID string `json:"serverGroupID"`
	ServiceName   string `json:"serviceName"`
	ServicePort   int    `json:"servicePort"`
	Weight        int    `json:"weight,omitempty"`
}
type MirrorGroupConfig struct {
	ServerGroupTuples []TrafficMirrorServerGroupTuple `json:"ServerGroupTuples" xml:"ServerGroupTuples"`
}
type TrafficLimitConfig struct {
	QPS      string `json:"QPS" xml:"QPS"`
	QPSPerIp string `json:"QPSPerIp" xml:"QPSPerIp"`
}
type FixedResponseConfig struct {
	Content     string `json:"Content" xml:"Content"`
	ContentType string `json:"ContentType" xml:"ContentType"`
	HttpCode    string `json:"HttpCode" xml:"HttpCode"`
}
type CorsConfig struct {
	AllowCredentials string   `json:"AllowCredentials" xml:"AllowCredentials"`
	MaxAge           string   `json:"MaxAge" xml:"MaxAge"`
	AllowOrigin      []string `json:"AllowOrigin" xml:"AllowOrigin"`
	AllowMethods     []string `json:"AllowMethods" xml:"AllowMethods"`
	AllowHeaders     []string `json:"AllowHeaders" xml:"AllowHeaders"`
	ExposeHeaders    []string `json:"ExposeHeaders" xml:"ExposeHeaders"`
}

type Condition struct {
	Type                     string                   `json:"Type" xml:"Type"`
	CookieConfig             CookieConfig             `json:"CookieConfig" xml:"CookieConfig"`
	HeaderConfig             HeaderConfig             `json:"HeaderConfig" xml:"HeaderConfig"`
	HostConfig               HostConfig               `json:"HostConfig" xml:"HostConfig"`
	MethodConfig             MethodConfig             `json:"MethodConfig" xml:"MethodConfig"`
	PathConfig               PathConfig               `json:"PathConfig" xml:"PathConfig"`
	QueryStringConfig        QueryStringConfig        `json:"QueryStringConfig" xml:"QueryStringConfig"`
	SourceIpConfig           SourceIpConfig           `json:"SourceIpConfig" xml:"SourceIpConfig"`
	ResponseStatusCodeConfig ResponseStatusCodeConfig `json:"ResponseStatusCodeConfig" xml:"ResponseStatusCodeConfig"`
	ResponseHeaderConfig     ResponseHeaderConfig     `json:"ResponseHeaderConfig" xml:"ResponseHeaderConfig"`
}
type CookieConfig struct {
	Values []Value `json:"Values" xml:"Values"`
}
type HeaderConfig struct {
	Key    string   `json:"Key" xml:"Key"`
	Values []string `json:"Values" xml:"Values"`
}
type Value struct {
	Key   string `json:"Key" xml:"Key"`
	Value string `json:"Value" xml:"Value"`
}
type HostConfig struct {
	Values []string `json:"Values" xml:"Values"`
}
type MethodConfig struct {
	Values []string `json:"Values" xml:"Values"`
}
type PathConfig struct {
	Values []string `json:"Values" xml:"Values"`
}
type QueryStringConfig struct {
	Values []Value `json:"Values" xml:"Values"`
}
type SourceIpConfig struct {
	Values []string `json:"Values" xml:"Values"`
}
type ResponseStatusCodeConfig struct {
	Values []string `json:"Values" xml:"Values"`
}
type ResponseHeaderConfig struct {
	Key    string   `json:"Key" xml:"Key"`
	Values []string `json:"Values" xml:"Values"`
}
