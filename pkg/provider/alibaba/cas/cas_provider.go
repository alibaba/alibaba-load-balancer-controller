package cas

import (
	"context"
	"sync"
	"time"

	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba/base"
	"k8s.io/alibaba-load-balancer-controller/pkg/util"
	"k8s.io/apimachinery/pkg/util/cache"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	cassdk "github.com/aliyun/alibaba-cloud-sdk-go/services/cas"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewCASProvider(
	auth *base.ClientMgr,
) *CASProvider {
	logger := ctrl.Log.WithName("controllers").WithName("CASProvider")
	return &CASProvider{
		auth:          auth,
		logger:        logger,
		loadCertMutex: &sync.Mutex{},
		certsCache:    cache.NewExpiring(),
		certsCacheTTL: 3 * time.Minute,
	}
}

var _ prvd.ICAS = &CASProvider{}

type CASProvider struct {
	auth          *base.ClientMgr
	logger        logr.Logger
	loadCertMutex *sync.Mutex
	certsCache    *cache.Expiring
	certsCacheTTL time.Duration
}

const (
	CASVersion  = "2021-06-19"
	CASDomain   = "cas.aliyuncs.com"
	CASShowSize = 50
)

const (
	certsCacheKey                         = "CertificateInfo"
	DescribeSSLCertificateList            = "DescribeSSLCertificateList"
	DescribeSSLCertificatePublicKeyDetail = "DescribeSSLCertificatePublicKeyDetail"
	CreateSSLCertificateWithName          = "CreateSSLCertificateWithName"
	DeleteSSLCertificate                  = "DeleteSSLCertificate"
	DefaultSSLCertificatePollInterval     = 30 * time.Second
	DefaultSSLCertificateTimeout          = 60 * time.Second
)

func (c CASProvider) DeleteSSLCertificate(ctx context.Context, certId string) error {
	traceID := ctx.Value(util.TraceID)
	req := cassdk.CreateDeleteSSLCertificateRequest()
	req.SetVersion(CASVersion)
	req.Domain = CASDomain
	req.CertIdentifier = certId
	var resp *cassdk.DeleteSSLCertificateResponse
	var err error
	if err := util.RetryImmediateOnError(DefaultSSLCertificatePollInterval, DefaultSSLCertificateTimeout, func(err error) bool {
		return false
	}, func() error {
		startTime := time.Now()
		c.logger.Info("deleting ssl certificate",
			"traceID", traceID,
			"startTime", startTime,
			"action", DeleteSSLCertificate)
		resp, err = c.auth.CAS.DeleteSSLCertificate(req)
		if err != nil {
			c.logger.Error(err, "DeleteSSLCertificate error")
			return err
		}
		c.logger.Info("deleted ssl certificate",
			"traceID", traceID,
			"CertIdentifier", req.CertIdentifier,
			"elapsedTime", time.Since(startTime).Milliseconds(),
			"requestID", resp.RequestId,
			"action", CreateSSLCertificateWithName)
		return nil
	}); err != nil {
		return errors.Wrap(err, "failed to deleteSSLCertificate")
	}
	c.loadCertMutex.Lock()
	defer c.loadCertMutex.Unlock()
	c.certsCache.Delete(certsCacheKey)
	return nil
}

func (c CASProvider) CreateSSLCertificateWithName(ctx context.Context, certName, certificate, privateKey string) (string, error) {
	traceID := ctx.Value(util.TraceID)

	req := cassdk.CreateCreateSSLCertificateWithNameRequest()
	req.SetVersion(CASVersion)
	req.Domain = CASDomain
	req.CertName = certName
	req.PrivateKey = privateKey
	req.Certificate = certificate
	var resp *cassdk.CreateSSLCertificateWithNameResponse
	var err error
	if err := util.RetryImmediateOnError(DefaultSSLCertificatePollInterval, DefaultSSLCertificateTimeout, func(err error) bool {
		return false
	}, func() error {
		startTime := time.Now()
		c.logger.Info("creating ssl certificate",
			"traceID", traceID,
			"startTime", startTime,
			"action", CreateSSLCertificateWithName)
		resp, err = c.auth.CAS.CreateSSLCertificateWithName(req)
		if err != nil {
			c.logger.Error(err, "CreateSSLCertificateWithName error")
			return err
		}
		c.logger.Info("created ssl certificate",
			"traceID", traceID,
			"certName", certName,
			"CertIdentifier", resp.CertIdentifier,
			"elapsedTime", time.Since(startTime).Milliseconds(),
			"requestID", resp.RequestId,
			"action", CreateSSLCertificateWithName)
		return nil
	}); err != nil {
		return "", errors.Wrap(err, "failed to createSSLCertificateWithName")
	}
	c.loadCertMutex.Lock()
	defer c.loadCertMutex.Unlock()
	c.certsCache.Delete(certsCacheKey)
	return resp.CertIdentifier, nil
}

func (c CASProvider) DescribeSSLCertificateList(ctx context.Context) ([]cassdk.CertificateInfo, error) {
	traceID := ctx.Value(util.TraceID)
	c.loadCertMutex.Lock()
	defer c.loadCertMutex.Unlock()

	if rawCacheItem, ok := c.certsCache.Get(certsCacheKey); ok {
		return rawCacheItem.([]cassdk.CertificateInfo), nil
	}

	req := cassdk.CreateDescribeSSLCertificateListRequest()
	req.SetVersion(CASVersion)
	req.Domain = CASDomain
	req.ShowSize = requests.NewInteger(CASShowSize)

	certificateInfos := make([]cassdk.CertificateInfo, 0)
	pageNumber := 1
	for {
		req.CurrentPage = requests.NewInteger(pageNumber)

		startTime := time.Now()
		c.logger.Info("listing ssl certificate",
			"traceID", traceID,
			"startTime", startTime,
			"action", DescribeSSLCertificateList)
		resp, err := c.auth.CAS.DescribeSSLCertificateList(req)
		if err != nil {
			c.logger.Error(err, "DescribeUserCertificateList error")
			return nil, err
		}
		c.logger.Info("listed ssl certificate",
			"traceID", traceID,
			"certMetaList", resp.CertMetaList,
			"elapsedTime", time.Since(startTime).Milliseconds(),
			"requestID", resp.RequestId,
			"action", DescribeSSLCertificateList)

		certificateInfos = append(certificateInfos, resp.CertMetaList...)

		if pageNumber < resp.PageCount {
			pageNumber++
		} else {
			break
		}
	}
	c.certsCache.Set(certsCacheKey, certificateInfos, c.certsCacheTTL)
	return certificateInfos, nil
}

func (c CASProvider) DescribeSSLCertificatePublicKeyDetail(ctx context.Context, request *cassdk.DescribeSSLCertificatePublicKeyDetailRequest) (*cassdk.DescribeSSLCertificatePublicKeyDetailResponse, error) {
	return c.auth.CAS.DescribeSSLCertificatePublicKeyDetail(request)
}
