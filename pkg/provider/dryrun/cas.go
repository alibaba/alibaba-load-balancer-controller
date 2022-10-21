package dryrun

import (
	"context"

	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	casprvd "k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/cas"

	cassdk "github.com/aliyun/alibaba-cloud-sdk-go/services/cas"
)

func NewDryRunCAS(
	auth *base.ClientMgr, cas *casprvd.CASProvider,
) *DryRunCAS {
	return &DryRunCAS{auth: auth, cas: cas}
}

var _ prvd.ICAS = &DryRunCAS{}

type DryRunCAS struct {
	auth *base.ClientMgr
	cas  *casprvd.CASProvider
}

func (c DryRunCAS) DeleteSSLCertificate(ctx context.Context, certId string) error {
	return nil
}
func (c DryRunCAS) CreateSSLCertificateWithName(ctx context.Context, certName, certificate, privateKey string) (string, error) {
	return "", nil
}

func (c DryRunCAS) DescribeSSLCertificateList(ctx context.Context) ([]cassdk.CertificateInfo, error) {
	return nil, nil
}

func (c DryRunCAS) DescribeSSLCertificatePublicKeyDetail(ctx context.Context, request *cassdk.DescribeSSLCertificatePublicKeyDetailRequest) (*cassdk.DescribeSSLCertificatePublicKeyDetailResponse, error) {
	return c.auth.CAS.DescribeSSLCertificatePublicKeyDetail(request)
}
