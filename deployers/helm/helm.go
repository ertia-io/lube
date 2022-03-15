package helm

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/storage/driver"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const helmOperationTimeout = 1800 * time.Second

type HelmDeployer struct {
	KubeConfig string
	RestConfig *restclient.Config
}

func NewHelmDeployer(kubeCfgPath string) (*HelmDeployer, error) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeCfgPath)
	if err != nil {
		return nil, fmt.Errorf("unable to load kubeconfig from %s: %v", kubeCfgPath, err)
	}

	return &HelmDeployer{
		KubeConfig: kubeCfgPath,
		RestConfig: config,
	}, nil
}

func (d *HelmDeployer) Name() string {
	return "HelmDeployer"
}

func (d *HelmDeployer) DeployPath(ctx context.Context, namespace string, path string) error {
	kubeCfg, err := ioutil.ReadFile(d.KubeConfig)
	if err != nil {
		return err
	}

	actionConfig := new(action.Configuration)
	restClientGetter := NewRESTClientGetter(namespace, string(kubeCfg))

	if err := actionConfig.Init(restClientGetter, namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return err
	}

	chart, err := loader.Load(path)
	if err != nil {
		return err
	}

	err = chart.Validate()
	if err != nil {
		return err
	}

	rn := releaseName(path)

	history := action.NewHistory(actionConfig)
	history.Max = 1

	if _, err := history.Run(rn); err == driver.ErrReleaseNotFound {
		installer := action.NewInstall(actionConfig)
		installer.Atomic = true
		installer.CreateNamespace = true
		installer.DependencyUpdate = true
		installer.IncludeCRDs = true
		installer.Namespace = namespace
		installer.ReleaseName = rn
		installer.Timeout = helmOperationTimeout

		_, err = installer.Run(chart, map[string]interface{}{})
		if err != nil {
			return err
		}

		return nil
	}

	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Atomic = true
	upgrade.Namespace = namespace
	upgrade.Timeout = helmOperationTimeout

	_, err = upgrade.Run(rn, chart, map[string]interface{}{})
	if err != nil {
		return err
	}

	return nil
}

// releaseName returns chart name from path, without file extention, as release name:
// /tmp/a/b/b/ertia-core.tgz -> ertia-core
func releaseName(path string) string {
	_, file := filepath.Split(path)
	ext := filepath.Ext(file)
	return strings.ToLower(file[0 : len(file)-len(ext)])
}
