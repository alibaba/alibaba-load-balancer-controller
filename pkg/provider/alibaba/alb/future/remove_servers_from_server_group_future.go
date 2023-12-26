package future

import (
	"fmt"
	"time"

	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
)

type RemoveServersFromServerGroupFuture struct {
	FutureBase
	ServerGroupId string
	Servers       []albsdk.RemoveServersFromServerGroupServers
	RequestId     string
	JobId         string
}

func NewRemoveServersFromServerGroupFuture(future FutureBase, serverGroupId string, servers []albsdk.RemoveServersFromServerGroupServers) *RemoveServersFromServerGroupFuture {
	return &RemoveServersFromServerGroupFuture{
		FutureBase:    future,
		ServerGroupId: serverGroupId,
		Servers:       servers,
	}
}
func (f *RemoveServersFromServerGroupFuture) Key() string {
	return f.FutureName
}

func (f *RemoveServersFromServerGroupFuture) Run() {
	removeServerFromSgpReq := albsdk.CreateRemoveServersFromServerGroupRequest()
	removeServerFromSgpReq.ServerGroupId = f.ServerGroupId
	removeServerFromSgpReq.Servers = &f.Servers

	startTime := time.Now()
	f.Logger.V(util.MgrLogLevel).Info("removing server from server group",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"servers", f.Servers,
		"startTime", startTime,
		util.Action, util.RemoveALBServersFromServerGroup)
	removeServerFromSgpResp, err := f.Client.RemoveServersFromServerGroup(removeServerFromSgpReq)
	if err != nil {
		close(f.Final)
		f.Success = false
		f.Err = err
		return
	}
	f.RequestId = removeServerFromSgpResp.RequestId
	f.JobId = removeServerFromSgpResp.JobId
	f.Logger.V(util.MgrLogLevel).Info("removed server from server group",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"requestID", f.RequestId,
		"elapsedTime", time.Since(startTime).Milliseconds(),
		util.Action, util.RemoveALBServersFromServerGroup)

}

func (f *RemoveServersFromServerGroupFuture) When() {
	if f.Err != nil {
		return
	}
	asynchronousStartTime := time.Now()
	f.Logger.V(util.MgrLogLevel).Info("removing server from server group asynchronous",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"servers", f.Servers,
		"startTime", asynchronousStartTime,
		util.Action, util.RemoveALBServersFromServerGroupAsynchronous)
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
			f.Logger.V(util.MgrLogLevel).Error(err, "failed to remove server from server group asynchronous",
				"serverGroupID", f.ServerGroupId,
				"traceID", f.TraceID,
				"requestID", f.RequestId,
				"elapsedTime", time.Since(asynchronousStartTime).Milliseconds(),
				util.Action, util.RemoveALBServersFromServerGroupAsynchronous)
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
		f.Logger.V(util.MgrLogLevel).Error(jobErr, "failed to remove server from server group asynchronous",
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
	f.Logger.V(util.MgrLogLevel).Info("removed server from server group asynchronous",
		"serverGroupID", f.ServerGroupId,
		"traceID", f.TraceID,
		"requestID", f.RequestId,
		"elapsedTime", time.Since(asynchronousStartTime).Milliseconds(),
		util.Action, util.RemoveALBServersFromServerGroupAsynchronous)
	close(f.Final)
	f.Success = true
}

func (f *RemoveServersFromServerGroupFuture) Result() {
	<-f.Final
}
