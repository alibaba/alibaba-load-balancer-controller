package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"

	"k8s.io/alibaba-load-balancer-controller/pkg/apis"
	"k8s.io/alibaba-load-balancer-controller/pkg/context/shared"
	prvd "k8s.io/alibaba-load-balancer-controller/pkg/provider"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/alibaba"
	"k8s.io/alibaba-load-balancer-controller/pkg/provider/dryrun"
	"k8s.io/alibaba-load-balancer-controller/version"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrl "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"k8s.io/alibaba-load-balancer-controller/cmd/health"
	ctrlCfg "k8s.io/alibaba-load-balancer-controller/pkg/config"
	"k8s.io/alibaba-load-balancer-controller/pkg/controller"
)

var log = klogr.New()

func printVersion() {
	log.Info(fmt.Sprintf("Alibaba Load Balancer Controller Version: %s, git commit: %s, build date: %s",
		version.Version, version.GitCommit, version.BuildDate))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	ctrl.SetLogger(klogr.New())
	printVersion()

	err := ctrlCfg.ControllerCFG.LoadControllerConfig()
	if err != nil {
		log.Error(err, "unable to load controller config")
		os.Exit(1)
	}
	ctrl.SetLogger(klogr.New().V(ctrlCfg.ControllerCFG.LogLevel))

	printVersion()

	// Get a config to talk to the api-server
	cfg := config.GetConfigOrDie()
	cfg.QPS = ctrlCfg.ControllerCFG.RuntimeConfig.QPS
	cfg.Burst = ctrlCfg.ControllerCFG.RuntimeConfig.Burst
	//cfg.ContentConfig = rest.ContentConfig{
	//	ContentType: "application/vnd.kubernetes.protobuf",
	//}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(cfg, ctrlCfg.BuildRuntimeOptions(ctrlCfg.ControllerCFG.RuntimeConfig))
	if err != nil {
		log.Error(err, "fail to create manager")
		os.Exit(1)
	}

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "add apis to schema: %s", err.Error())
		os.Exit(1)
	}

	var cloud prvd.Provider
	if ctrlCfg.ControllerCFG.DryRun {
		log.Info("using DryRun Mode")
		cloud = dryrun.NewDryRunCloud()
	} else {
		cloud = alibaba.NewAlibabaCloud()
	}
	log.Info("Creating context.")
	ctx := shared.NewSharedContext(cloud)

	log.Info("Registering Components.")
	if err := controller.AddToManager(mgr, ctx, ctrlCfg.ControllerCFG.Controllers); err != nil {
		log.Error(err, "add controller: %s", err.Error())
		os.Exit(1)
	} else {
		log.Info(fmt.Sprintf("Loaded controllers: %v", ctrlCfg.ControllerCFG.Controllers))
	}

	// Start the Cmd
	log.Info("Starting the Cmd.")
	if err := mgr.AddHealthzCheck("default", func(req *http.Request) error {
		errs := make([]error, 0)
		for _, fun := range health.CheckFuncList {
			if err := fun.Check(); err != nil {
				errs = append(errs, err)
			}
		}
		return utilerrors.NewAggregate(errs)
	}); err != nil {
		log.Error(err, "failed to add default health check: %w", err.Error())
		os.Exit(1)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero: %s", err.Error())
		os.Exit(1)
	}

}
