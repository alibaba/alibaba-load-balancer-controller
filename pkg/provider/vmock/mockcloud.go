package vmock

import (
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
)

func NewMockCloud(auth *base.ClientMgr) prvd.Provider {

	return &MockCloud{
		IMetaData: auth.Meta,
		MockECS:   NewMockECS(auth),
		MockVPC:   NewMockVPC(auth),
		MockALB:   NewMockALB(auth),
		MockSLS:   NewMockSLS(auth),
		MockCAS:   NewMockCAS(auth),
		MockNLB:   NewMockNLB(auth),
	}
}

var _ prvd.Provider = alibaba.AlibabaCloud{}

// MockCloud for unit test
type MockCloud struct {
	*MockECS
	*MockVPC
	*MockALB
	*MockCAS
	*MockSLS
	*MockNLB
	prvd.IMetaData
}
