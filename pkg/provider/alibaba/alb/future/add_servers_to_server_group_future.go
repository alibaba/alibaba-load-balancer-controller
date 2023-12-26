package future

import (
	"fmt"
	"time"

	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
)

type AddServersToServerGroupFuture struct {
	FutureBase
	ServerGroupId string
	Servers       []albsdk.AddServersToServerGroupServers
	RequestId     string
	JobId         string
}

func NewAddServersToServerGroupFuture(future FutureBase, serverGroupId string, servers []albsdk.AddServersToServerGroupServers) *AddServersToServerGroupFuture {
	return &AddServersToServerGroupFuture{
		FutureBase:    future,
		ServerGroupId: serverGroupId,
		Servers:       servers,
	}
}

func (f *AddServersToServerGroupFuture) Key() string {
	return f.FutureName
}

func (f *AddServersToServerGroupFuture) Run() {

	addServerToSgpReq := albsdk.CreateAddServersToServerGroupRequest()
	addServerToSgpReq.ServerGroupId = f.ServerGroupId
	addServerToSgpReq.Servers = &f.Servers

	startTime := time.Now()
	f.Logger.V(util.MgrLogLevel).Info("adding server to server group",
		"serverGroupID", f.ServerGroupId,
		"servers", f.Servers,
		"traceID", f.TraceID,
		"startTime", startTime,
		util.Action, util.AddALBServersToServerGroup)
	addServerToSgpResp, err := f.Client.AddServersToServerGroup(addServerToSgpReq)
	if err != nil {
		close(f.Final)
		f.Success = false
		f.Err = err
		return
	}
	f.Logger.V(util.MgrLogLevel).Info("added server to server group",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"requestID", addServerToSgpResp.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.AddALBServersToServerGroup)

	f.RequestId = addServerToSgpResp.RequestId
	f.JobId = addServerToSgpResp.JobId

}

func (f *AddServersToServerGroupFuture) When() {
	if f.Err != nil {
		return
	}
	asynchronousStartTime := time.Now()
	f.Logger.V(util.MgrLogLevel).Info("adding server to server group asynchronous",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"servers", f.Servers,
		"startTime", asynchronousStartTime,
		util.Action, util.AddALBServersToServerGroupAsynchronous)
	var jobErr error
	var success bool
	asyncJobReq := albsdk.CreateListAsynJobsRequest()
	jobs := []string{f.JobId}
	asyncJobReq.JobIds = &jobs

	for i := 0; i < util.RemoveALBServersFromServerGroupMaxRetryTimes; i++ {
		time.Sleep(util.RemoveALBServersFromServerGroupRetryInterval)
		asyncJobResp, err := f.Client.ListAsynJobs(asyncJobReq)
		if err == nil && asyncJobResp.TotalCount != 1 {
			err = fmt.Errorf("incorrect ListAsynJobs number: %d", asyncJobResp.TotalCount)
		}
		if err != nil {
			f.Logger.V(util.MgrLogLevel).Error(err, "failed adding server to server group asynchronous",
				"serverGroupID", f.ServerGroupId,
				"traceID", f.TraceID,
				"requestID", f.RequestId,
				"elapsedTime", time.Since(asynchronousStartTime).Milliseconds(),
				util.Action, util.AddALBServersToServerGroupAsynchronous)
		}
		job := asyncJobResp.Jobs[0]
		if util.ListAsynJobsStatusProcessing == job.Status {
			continue
		}
		if util.ListAsynJobsStatusSucceeded == job.Status {
			success = true
		}
		if util.ListAsynJobsStatusFailed == job.Status {
			jobErr = fmt.Errorf(job.ErrorMessage)
		}
		break
	}
	if jobErr == nil && !success {
		jobErr = fmt.Errorf("wait asyncJob status timeout")
	}
	if jobErr != nil {
		f.Logger.V(util.MgrLogLevel).Error(jobErr, "failed adding server to server group asynchronous",
			"serverGroupID", f.ServerGroupId,
			"traceID", f.TraceID,
			"requestID", f.RequestId,
			"elapsedTime", time.Since(asynchronousStartTime).Milliseconds(),
			util.Action, util.RemoveALBServersFromServerGroupAsynchronous)
		close(f.Final)
		f.Success = false
		f.Err = jobErr
		return
	}
	f.Logger.V(util.MgrLogLevel).Info("added server to server group asynchronous",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"requestID", f.RequestId,
		"elapsedTime", time.Since(asynchronousStartTime).Milliseconds(),
		util.Action, util.RemoveALBServersFromServerGroupAsynchronous)
	close(f.Final)
	f.Success = true
}

func (f *AddServersToServerGroupFuture) Result() {
	<-f.Final
}
