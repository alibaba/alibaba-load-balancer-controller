package alb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/alb/future"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb"
)

var registerServersFunc = func(ctx context.Context, serverMgr *ALBProvider, sgpID string, servers []albsdk.AddServersToServerGroupServers) error {
	if len(servers) == 0 {
		return nil
	}

	traceID := ctx.Value(util.TraceID)
	future := future.NewAddServersToServerGroupFuture(future.NewFutureBase(util.AddALBServersToServerGroup,
		traceID,
		serverMgr.auth.ALB,
		serverMgr.logger),
		sgpID, servers)

	serverMgr.promise.Start(future)
	if !future.Success {
		return future.Err
	}
	return nil
}

type BatchRegisterServersFunc func(context.Context, *ALBProvider, string, []albsdk.AddServersToServerGroupServers) error

func BatchRegisterServers(ctx context.Context, serverMgr *ALBProvider, sgpID string, servers []albsdk.AddServersToServerGroupServers, cnt int, batch BatchRegisterServersFunc) error {
	if cnt <= 0 || cnt >= util.BatchRegisterDeregisterServersMaxNum {
		cnt = util.BatchRegisterServersDefaultNum
	}

	for len(servers) > cnt {
		if err := batch(ctx, serverMgr, sgpID, servers[0:cnt]); err != nil {
			return err
		}
		servers = servers[cnt:]
	}
	if len(servers) <= 0 {
		return nil
	}

	return batch(ctx, serverMgr, sgpID, servers)
}

func (m *ALBProvider) RegisterALBServers(ctx context.Context, serverGroupID string, resServers []alb.BackendItem) error {
	if len(serverGroupID) == 0 {
		return fmt.Errorf("empty server group id when register servers error")
	}

	if len(resServers) == 0 {
		return nil
	}

	serversToAdd, err := transModelBackendsToSDKAddServersToServerGroupServers(resServers)
	if err != nil {
		return err
	}

	return BatchRegisterServers(ctx, m, serverGroupID, serversToAdd, util.BatchRegisterServersDefaultNum, registerServersFunc)
}

var deregisterServersFunc = func(ctx context.Context, serverMgr *ALBProvider, sgpID string, servers []albsdk.RemoveServersFromServerGroupServers) error {
	if len(servers) == 0 {
		return nil
	}

	traceID := ctx.Value(util.TraceID)
	future := future.NewRemoveServersFromServerGroupFuture(future.NewFutureBase(util.RemoveALBServersFromServerGroup,
		traceID,
		serverMgr.auth.ALB,
		serverMgr.logger),
		sgpID, servers)

	serverMgr.promise.Start(future)
	if !future.Success {
		return future.Err
	}
	return nil
}

type DeregisterServersFunc func(context.Context, *ALBProvider, string, []albsdk.RemoveServersFromServerGroupServers) error

func BatchDeregisterServers(ctx context.Context, serverMgr *ALBProvider, sgpID string, servers []albsdk.RemoveServersFromServerGroupServers, cnt int, batch DeregisterServersFunc) error {
	if cnt <= 0 || cnt >= util.BatchRegisterDeregisterServersMaxNum {
		cnt = util.BatchRegisterServersDefaultNum
	}

	for len(servers) > cnt {
		if err := batch(ctx, serverMgr, sgpID, servers[0:cnt]); err != nil {
			return err
		}
		servers = servers[cnt:]
	}
	if len(servers) <= 0 {
		return nil
	}

	return batch(ctx, serverMgr, sgpID, servers)
}

func (m *ALBProvider) DeregisterALBServers(ctx context.Context, serverGroupID string, sdkServers []albsdk.BackendServer) error {
	if len(serverGroupID) == 0 {
		return fmt.Errorf("empty server group id when deregister servers error")
	}

	if len(sdkServers) == 0 {
		return nil
	}

	serversToRemove := make([]albsdk.RemoveServersFromServerGroupServers, 0)
	for _, sdkServer := range sdkServers {
		if isServerStatusRemoving(sdkServer.Status) {
			continue
		}
		serverToRemove, err := transSDKBackendServerToRemoveServersFromServerGroupServer(sdkServer)
		if err != nil {
			return err
		}
		serversToRemove = append(serversToRemove, *serverToRemove)
	}

	if len(serversToRemove) == 0 {
		return nil
	}

	return BatchDeregisterServers(ctx, m, serverGroupID, serversToRemove, util.BatchDeregisterServersDefaultNum, deregisterServersFunc)
}

func (m *ALBProvider) ReplaceALBServers(ctx context.Context, serverGroupID string, resServers []alb.BackendItem, sdkServers []albsdk.BackendServer) error {
	if len(serverGroupID) == 0 {
		return fmt.Errorf("empty server group id when replace servers error")
	}

	traceID := ctx.Value(util.TraceID)

	if len(resServers) == 0 && len(sdkServers) == 0 {
		return nil
	}

	addedServers, err := transModelBackendsToSDKReplaceServersInServerGroupAddedServers(resServers)
	if err != nil {
		return err
	}

	removedServers := make([]albsdk.ReplaceServersInServerGroupRemovedServers, 0)
	for _, sdkServer := range sdkServers {
		if isServerStatusRemoving(sdkServer.Status) {
			continue
		}
		serverToRemove, err := transSDKBackendServerToReplaceServersInServerGroupRemovedServer(sdkServer)
		if err != nil {
			return err
		}
		removedServers = append(removedServers, *serverToRemove)
	}

	replaceServerFromSgpReq := albsdk.CreateReplaceServersInServerGroupRequest()
	replaceServerFromSgpReq.ServerGroupId = serverGroupID
	replaceServerFromSgpReq.AddedServers = &addedServers
	replaceServerFromSgpReq.RemovedServers = &removedServers

	startTime := time.Now()
	m.logger.V(util.MgrLogLevel).Info("replacing server in server group",
		"serverGroupID", serverGroupID,
		"traceID", traceID,
		"addedServers", addedServers,
		"removedServers", removedServers,
		"startTime", startTime,
		util.Action, util.ReplaceALBServersInServerGroup)
	replaceServerFromSgpResp, err := m.auth.ALB.ReplaceServersInServerGroup(replaceServerFromSgpReq)
	if err != nil {
		return err
	}
	m.logger.V(util.MgrLogLevel).Info("replaced server in server group",
		"serverGroupID", serverGroupID,
		"traceID", traceID,
		"requestID", replaceServerFromSgpResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.ReplaceALBServersInServerGroup)

	if util.IsWaitServersAsynchronousComplete {
		asynchronousStartTime := time.Now()
		m.logger.V(util.MgrLogLevel).Info("replacing server in server group asynchronous",
			"serverGroupID", serverGroupID,
			"traceID", traceID,
			"addedServers", addedServers,
			"removedServers", removedServers,
			"startTime", startTime,
			util.Action, util.ReplaceALBServersInServerGroupAsynchronous)
		for i := 0; i < util.ReplaceALBServersInServerGroupMaxRetryTimes; i++ {
			time.Sleep(util.ReplaceALBServersInServerGroupRetryInterval)

			isCompleted, err := isReplaceServersCompleted(ctx, m, serverGroupID, addedServers, removedServers)
			if err != nil {
				m.logger.V(util.MgrLogLevel).Error(err, "failed to replace server in server group asynchronous",
					"serverGroupID", serverGroupID,
					"traceID", traceID,
					"requestID", replaceServerFromSgpResp.RequestId,
					util.Action, util.ReplaceALBServersInServerGroupAsynchronous)
				return err
			}
			if isCompleted {
				break
			}
		}
		m.logger.V(util.MgrLogLevel).Info("replaced server in server group asynchronous",
			"serverGroupID", serverGroupID,
			"traceID", traceID,
			"requestID", replaceServerFromSgpResp.RequestId,
			"elapsedTime", time.Since(asynchronousStartTime).Milliseconds(),
			util.Action, util.ReplaceALBServersInServerGroupAsynchronous)
	}

	return nil
}

func (m *ALBProvider) ListALBServers(ctx context.Context, serverGroupID string) ([]albsdk.BackendServer, error) {
	if len(serverGroupID) == 0 {
		return nil, fmt.Errorf("empty server group id when list servers error")
	}

	traceID := ctx.Value(util.TraceID)

	var (
		nextToken string
		servers   []albsdk.BackendServer
	)

	for {
		future := future.NewListServerGroupServersFuture(future.NewFutureBase(util.ListALBServerGroupServers,
			traceID,
			m.auth.ALB,
			m.logger), serverGroupID, nextToken)

		m.promise.Start(future)
		if !future.Success {
			return nil, future.Err
		}
		servers = append(servers, future.Servers...)

		if future.NextToken == "" {
			break
		} else {
			nextToken = future.NextToken
		}
	}

	return servers, nil
}

func isServerStatusRemoving(status string) bool {
	return strings.EqualFold(status, util.ServerStatusRemoving)
}

func transSDKBackendServerToRemoveServersFromServerGroupServer(server albsdk.BackendServer) (*albsdk.RemoveServersFromServerGroupServers, error) {
	serverToRemove := new(albsdk.RemoveServersFromServerGroupServers)

	serverToRemove.ServerIp = server.ServerIp
	if len(server.ServerId) == 0 {
		return nil, fmt.Errorf("invalid server id for server: %v", server)
	}
	serverToRemove.ServerId = server.ServerId

	if !isServerPortValid(server.Port) {
		return nil, fmt.Errorf("invalid server port for server: %v", server)
	}
	serverToRemove.Port = strconv.Itoa(server.Port)

	if !isServerTypeValid(server.ServerType) {
		return nil, fmt.Errorf("invalid server type for server: %v", server)
	}
	serverToRemove.ServerType = server.ServerType

	return serverToRemove, nil
}

func transSDKBackendServerToReplaceServersInServerGroupRemovedServer(server albsdk.BackendServer) (*albsdk.ReplaceServersInServerGroupRemovedServers, error) {
	serverToRemove := new(albsdk.ReplaceServersInServerGroupRemovedServers)

	serverToRemove.ServerIp = server.ServerIp
	if len(server.ServerId) == 0 {
		return nil, fmt.Errorf("invalid server id for server: %v", server)
	}
	serverToRemove.ServerId = server.ServerId

	if !isServerPortValid(server.Port) {
		return nil, fmt.Errorf("invalid server port for server: %v", server)
	}
	serverToRemove.Port = strconv.Itoa(server.Port)

	if !isServerTypeValid(server.ServerType) {
		return nil, fmt.Errorf("invalid server type for server: %v", server)
	}
	serverToRemove.ServerType = server.ServerType

	return serverToRemove, nil
}

func transModelBackendToSDKAddServersToServerGroupServer(server alb.BackendItem) (*albsdk.AddServersToServerGroupServers, error) {
	serverToAdd := new(albsdk.AddServersToServerGroupServers)

	serverToAdd.ServerIp = server.ServerIp

	if len(server.ServerId) == 0 {
		return nil, fmt.Errorf("invalid server id for server: %v", server)
	}
	serverToAdd.ServerId = server.ServerId

	if !isServerPortValid(server.Port) {
		return nil, fmt.Errorf("invalid server port for server: %v", server)
	}
	serverToAdd.Port = strconv.Itoa(server.Port)

	if !isServerTypeValid(server.Type) {
		return nil, fmt.Errorf("invalid server type for server: %v", server)
	}
	serverToAdd.ServerType = server.Type

	if !isServerWeightValid(server.Weight) {
		return nil, fmt.Errorf("invalid server weight for server: %v", server)
	}
	serverToAdd.Weight = strconv.Itoa(server.Weight)

	return serverToAdd, nil
}

func transModelBackendToSDKReplaceServersInServerGroupAddedServer(server alb.BackendItem) (*albsdk.ReplaceServersInServerGroupAddedServers, error) {
	serverToAdd := new(albsdk.ReplaceServersInServerGroupAddedServers)

	serverToAdd.ServerIp = server.ServerIp

	if len(server.ServerId) == 0 {
		return nil, fmt.Errorf("invalid server id for server: %v", server)
	}
	serverToAdd.ServerId = server.ServerId

	if !isServerPortValid(server.Port) {
		return nil, fmt.Errorf("invalid server port for server: %v", server)
	}
	serverToAdd.Port = strconv.Itoa(server.Port)

	if !isServerTypeValid(server.Type) {
		return nil, fmt.Errorf("invalid server type for server: %v", server)
	}
	serverToAdd.ServerType = server.Type

	if !isServerWeightValid(server.Weight) {
		return nil, fmt.Errorf("invalid server weight for server: %v", server)
	}
	serverToAdd.Weight = strconv.Itoa(server.Weight)

	return serverToAdd, nil
}

func transModelBackendsToSDKAddServersToServerGroupServers(servers []alb.BackendItem) ([]albsdk.AddServersToServerGroupServers, error) {
	serversToAdd := make([]albsdk.AddServersToServerGroupServers, 0)
	for _, resServer := range servers {
		serverToAdd, err := transModelBackendToSDKAddServersToServerGroupServer(resServer)
		if err != nil {
			return nil, err
		}
		serversToAdd = append(serversToAdd, *serverToAdd)
	}
	return serversToAdd, nil
}

func transModelBackendsToSDKReplaceServersInServerGroupAddedServers(servers []alb.BackendItem) ([]albsdk.ReplaceServersInServerGroupAddedServers, error) {
	serversToAdd := make([]albsdk.ReplaceServersInServerGroupAddedServers, 0)
	for _, resServer := range servers {
		serverToAdd, err := transModelBackendToSDKReplaceServersInServerGroupAddedServer(resServer)
		if err != nil {
			return nil, err
		}
		serversToAdd = append(serversToAdd, *serverToAdd)
	}
	return serversToAdd, nil
}

func isServerPortValid(port int) bool {
	if port < 1 || port > 65535 {
		return false
	}
	return true
}

func isServerTypeValid(serverType string) bool {
	if !strings.EqualFold(serverType, util.ServerTypeEcs) &&
		!strings.EqualFold(serverType, util.ServerTypeEni) &&
		!strings.EqualFold(serverType, util.ServerTypeEci) {
		return false
	}

	return true
}

func isServerWeightValid(weight int) bool {
	if weight < 0 || weight > 100 {
		return false
	}
	return true
}

func isRegisterServersForReplaceCompleted(sdkServers []albsdk.BackendServer, servers []albsdk.ReplaceServersInServerGroupAddedServers) (bool, error) {
	var isCompleted = true
	for _, server := range servers {
		var serverUID string
		if len(server.ServerIp) == 0 {
			serverUID = fmt.Sprintf("%v:%v", server.ServerId, server.Port)
		} else {
			serverUID = fmt.Sprintf("%v:%v:%v", server.ServerId, server.ServerIp, server.Port)
		}

		isExist := false
		var backendServer albsdk.BackendServer
		for _, sdkServer := range sdkServers {
			var sdkServerUID string
			if len(server.ServerIp) == 0 {
				sdkServerUID = fmt.Sprintf("%v:%v", sdkServer.ServerId, sdkServer.Port)
			} else {
				sdkServerUID = fmt.Sprintf("%v:%v:%v", sdkServer.ServerId, sdkServer.ServerIp, sdkServer.Port)
			}
			if strings.EqualFold(serverUID, sdkServerUID) {
				isExist = true
				backendServer = sdkServer
				break
			}
		}

		if isExist && strings.EqualFold(backendServer.Status, util.ServerStatusAvailable) {
			continue
		}

		isCompleted = false
		break
	}

	if isCompleted {
		return true, nil
	}

	return false, nil
}

func isDeregisterServersForReplaceCompleted(sdkServers []albsdk.BackendServer, servers []albsdk.ReplaceServersInServerGroupRemovedServers) (bool, error) {
	var isCompleted = true
	for _, server := range servers {
		var serverUID string
		if len(server.ServerIp) == 0 {
			serverUID = fmt.Sprintf("%v:%v", server.ServerId, server.Port)
		} else {
			serverUID = fmt.Sprintf("%v:%v:%v", server.ServerId, server.ServerIp, server.Port)
		}

		isExist := false
		for _, sdkServer := range sdkServers {
			var sdkServerUID string
			if len(server.ServerIp) == 0 {
				sdkServerUID = fmt.Sprintf("%v:%v", sdkServer.ServerId, sdkServer.Port)
			} else {
				sdkServerUID = fmt.Sprintf("%v:%v:%v", sdkServer.ServerId, sdkServer.ServerIp, sdkServer.Port)
			}
			if strings.EqualFold(serverUID, sdkServerUID) {
				isExist = true
				break
			}
		}

		if isExist {
			isCompleted = false
			break
		}
	}

	if isCompleted {
		return true, nil
	}

	return false, nil
}

func isReplaceServersCompleted(ctx context.Context, m *ALBProvider, serverGroupID string, registerServers []albsdk.ReplaceServersInServerGroupAddedServers, deregisterServers []albsdk.ReplaceServersInServerGroupRemovedServers) (bool, error) {
	sdkServers, err := m.ListALBServers(ctx, serverGroupID)
	if err != nil {
		return false, err
	}

	isRegisterComplete, err := isRegisterServersForReplaceCompleted(sdkServers, registerServers)
	if err != nil {
		return false, err
	}
	isDeregisterComplete, err := isDeregisterServersForReplaceCompleted(sdkServers, deregisterServers)
	if err != nil {
		return false, err
	}

	if isRegisterComplete && isDeregisterComplete {
		return true, nil
	}

	return false, nil

}
