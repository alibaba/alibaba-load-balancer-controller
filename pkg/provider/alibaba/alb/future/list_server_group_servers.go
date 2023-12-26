package future

import (
	"time"

	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
)

type ListServerGroupServersFuture struct {
	FutureBase
	ServerGroupId string
	Token         string
	NextToken     string
	Servers       []albsdk.BackendServer
	RequestId     string
}

func NewListServerGroupServersFuture(future FutureBase, serverGroupId, token string) *ListServerGroupServersFuture {
	return &ListServerGroupServersFuture{
		FutureBase:    future,
		ServerGroupId: serverGroupId,
		Token:         token,
	}
}

func (f *ListServerGroupServersFuture) Key() string {
	return f.FutureName
}

func (f *ListServerGroupServersFuture) Run() {

	listSgpServersReq := albsdk.CreateListServerGroupServersRequest()
	listSgpServersReq.ServerGroupId = f.ServerGroupId
	listSgpServersReq.NextToken = f.Token

	startTime := time.Now()
	f.Logger.V(util.MgrLogLevel).Info("listing servers",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"startTime", startTime,
		util.Action, util.ListALBServerGroupServers)
	listSgpServersResp, err := f.Client.ListServerGroupServers(listSgpServersReq)
	if err != nil {
		close(f.Final)
		f.Success = false
		f.Err = err
		return
	}
	f.Logger.V(util.MgrLogLevel).Info("listed servers",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"servers", listSgpServersResp.Servers,
		"requestID", listSgpServersResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.ListALBServerGroupServers)

	f.RequestId = listSgpServersResp.RequestId
	f.Servers = listSgpServersResp.Servers
	f.NextToken = listSgpServersResp.NextToken
}

func (f *ListServerGroupServersFuture) When() {
	if f.Err != nil {
		return
	}
	f.Success = true
	close(f.Final)
}

func (f *ListServerGroupServersFuture) Result() {
	<-f.Final
}
