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
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

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

	// TODO: Update with install if firts run e.g.:
	// helm upgrade --install --atomic ...
	installer := action.NewInstall(actionConfig)
	installer.Atomic = true
	installer.CreateNamespace = true
	installer.DependencyUpdate = true
	installer.IncludeCRDs = true
	installer.Namespace = namespace
	installer.ReleaseName = releaseName(path)
	installer.Timeout = time.Second * 1800

	chart, err := loader.Load(path)
	if err != nil {
		return err
	}

	err = chart.Validate()
	if err != nil {
		return err
	}

	_, err = installer.Run(chart, map[string]interface{}{})
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
