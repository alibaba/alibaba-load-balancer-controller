package ess

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// ModifyScheduledTask invokes the ess.ModifyScheduledTask API synchronously
func (client *Client) ModifyScheduledTask(request *ModifyScheduledTaskRequest) (response *ModifyScheduledTaskResponse, err error) {
	response = CreateModifyScheduledTaskResponse()
	err = client.DoAction(request, response)
	return
}

// ModifyScheduledTaskWithChan invokes the ess.ModifyScheduledTask API asynchronously
func (client *Client) ModifyScheduledTaskWithChan(request *ModifyScheduledTaskRequest) (<-chan *ModifyScheduledTaskResponse, <-chan error) {
	responseChan := make(chan *ModifyScheduledTaskResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.ModifyScheduledTask(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// ModifyScheduledTaskWithCallback invokes the ess.ModifyScheduledTask API asynchronously
func (client *Client) ModifyScheduledTaskWithCallback(request *ModifyScheduledTaskRequest, callback func(response *ModifyScheduledTaskResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *ModifyScheduledTaskResponse
		var err error
		defer close(result)
		response, err = client.ModifyScheduledTask(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// ModifyScheduledTaskRequest is the request struct for api ModifyScheduledTask
type ModifyScheduledTaskRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	ScheduledAction      string           `position:"Query" name:"ScheduledAction"`
	MaxValue             requests.Integer `position:"Query" name:"MaxValue"`
	ScalingGroupId       string           `position:"Query" name:"ScalingGroupId"`
	Description          string           `position:"Query" name:"Description"`
	RecurrenceEndTime    string           `position:"Query" name:"RecurrenceEndTime"`
	LaunchTime           string           `position:"Query" name:"LaunchTime"`
	DesiredCapacity      requests.Integer `position:"Query" name:"DesiredCapacity"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerAccount         string           `position:"Query" name:"OwnerAccount"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
	RecurrenceValue      string           `position:"Query" name:"RecurrenceValue"`
	LaunchExpirationTime requests.Integer `position:"Query" name:"LaunchExpirationTime"`
	MinValue             requests.Integer `position:"Query" name:"MinValue"`
	ScheduledTaskName    string           `position:"Query" name:"ScheduledTaskName"`
	TaskEnabled          requests.Boolean `position:"Query" name:"TaskEnabled"`
	ScheduledTaskId      string           `position:"Query" name:"ScheduledTaskId"`
	RecurrenceType       string           `position:"Query" name:"RecurrenceType"`
}

// ModifyScheduledTaskResponse is the response struct for api ModifyScheduledTask
type ModifyScheduledTaskResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateModifyScheduledTaskRequest creates a request to invoke ModifyScheduledTask API
func CreateModifyScheduledTaskRequest() (request *ModifyScheduledTaskRequest) {
	request = &ModifyScheduledTaskRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ess", "2014-08-28", "ModifyScheduledTask", "ess", "openAPI")
	request.Method = requests.POST
	return
}

// CreateModifyScheduledTaskResponse creates a response to parse from ModifyScheduledTask response
func CreateModifyScheduledTaskResponse() (response *ModifyScheduledTaskResponse) {
	response = &ModifyScheduledTaskResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
