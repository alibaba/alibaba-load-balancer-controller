package future

import (
	albsdk "github.com/aliyun/alibaba-cloud-sdk-go/services/alb"
	"github.com/go-logr/logr"
)

type FutureBase struct {
	FutureName string
	TraceID    any
	Client     *albsdk.Client
	Logger     logr.Logger
	Final      chan struct{}
	Success    bool
	Err        error
}

func NewFutureBase(futureName string, traceID any, client *albsdk.Client, logger logr.Logger) FutureBase {
	return FutureBase{
		FutureName: futureName,
		TraceID:    traceID,
		Client:     client,
		Logger:     logger,
		Final:      make(chan struct{}),
		Success:    false,
		Err:        nil,
	}
}

type Future interface {
	Key() string
	Run()
	When()
	Result()
}
