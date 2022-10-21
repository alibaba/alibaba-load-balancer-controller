package framework

import (
	"k8s.io/alibaba-load-balancer-controller/test/e2e/client"
	"k8s.io/alibaba-load-balancer-controller/test/e2e/options"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type Framework struct {
	Client          *client.E2EClient
	CreatedResource map[string]string
}

func NewFrameWork(c *client.E2EClient) *Framework {
	return &Framework{
		Client:          c,
		CreatedResource: make(map[string]string, 0),
	}
}

func (f *Framework) BeforeSuit() error {
	err := f.Client.KubeClient.CreateNamespace()
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = f.Client.KubeClient.DeleteService()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if err := f.Client.KubeClient.CreateDeployment(); err != nil {
		return err
	}

	if options.TestConfig.EnableVK {
		if err := f.Client.KubeClient.CreateVKDeployment(); err != nil {
			return err
		}
	}

	return nil
}

func (f *Framework) AfterSuit() error {
	err := f.Client.KubeClient.DeleteNamespace()
	if err != nil {
		return err
	}
	return nil
}

func (f *Framework) AfterEach() error {
	_, err := f.Client.KubeClient.GetService()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err = f.Client.KubeClient.DeleteService(); err != nil {
		return err
	}
	return nil
}
