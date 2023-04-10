package controller

import (
	"fmt"

	"k8s.io/alibaba-load-balancer-controller/pkg/context/shared"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller/ingress"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller/service"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	controllerMap = map[string]func(manager.Manager, *shared.SharedContext) error{
		"ingress": ingress.Add,
		"service": service.Add,
	}
}

// ControllerMap is a list of functions to add all Controllers to the Manager
var controllerMap map[string]func(manager.Manager, *shared.SharedContext) error

// AddToManager adds selected Controllers to the Manager
func AddToManager(m manager.Manager, ctx *shared.SharedContext, enableControllers []string) error {
	for _, cont := range enableControllers {
		if f, ok := controllerMap[cont]; ok {
			if err := f(m, ctx); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot find controller %s", cont)
		}
	}
	return nil
}
