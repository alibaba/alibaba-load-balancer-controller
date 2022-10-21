package e2e

import (
	"strings"
	"testing"

	"k8s.io/alibaba-load-balancer-controller/pkg/util"

	"k8s.io/alibaba-load-balancer-controller/test/e2e/client"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/framework"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/options"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/testcase/alb"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/test/e2e/framework/ginkgowrapper"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

func init() {
	options.TestConfig.BindFlags()
}

func TestE2E(t *testing.T) {
	err := options.TestConfig.Validate()
	if err != nil {
		t.Fatalf("test config validate failed: %s", err.Error())
	}

	c, err := client.NewClient()
	if err != nil {
		t.Fatalf("create client error: %s", err.Error())
	}
	f := framework.NewFrameWork(c)
	klog.Infof("test config: %s", util.PrettyJson(options.TestConfig))

	gomega.RegisterFailHandler(ginkgowrapper.Fail)

	ginkgo.Describe("Run cloud controller manager e2e tests", func() {

		ginkgo.BeforeSuite(func() {
			err = f.BeforeSuit()
			gomega.Expect(err).To(gomega.BeNil())
		})

		ginkgo.AfterSuite(func() {
			err = f.AfterSuit()
			gomega.Expect(err).To(gomega.BeNil())
		})

		AddControllerTests(f)
	})

	ginkgo.RunSpecs(t, "run ccm e2e test")
}

func AddControllerTests(f *framework.Framework) {
	controllers := strings.Split(options.TestConfig.Controllers, ",")

	klog.Infof("test controllers: %s", options.TestConfig.Controllers)
	if len(controllers) == 0 {
		klog.Info("no controller tests need to run, finished")
		return
	}
	for _, c := range controllers {
		switch c {
		case "alb":
			albFlags := strings.Split(options.TestConfig.ALBFlags, ",")
			ginkgo.Describe("alb ingress controller tests", func() {
				alb.RunAlbConfigTestCases(f)
				alb.InitAlbIngressE2ECases()
				alb.ExecuteIngressE2ECases(f, albFlags)
				alb.CleanAlbconfigTestCases(f)
			})
		default:
			klog.Infof("%s controller is not supported", c)
		}

	}
}
