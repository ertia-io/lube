package helm

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/storage/driver"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	helmOperationTimeout = 3600 * time.Second
	helmValuesFile       = "values.yaml"

	placeholderDomain = `@DOMAIN@`
)

type HelmDeployer struct {
	Debug      bool
	Domain     string
	KubeConfig string
	RestConfig *restclient.Config
}

func NewHelmDeployer(kubeCfgPath, domain string) (*HelmDeployer, error) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeCfgPath)
	if err != nil {
		return nil, fmt.Errorf("unable to load kubeconfig from %s: %v", kubeCfgPath, err)
	}

	return &HelmDeployer{
		Domain:     domain,
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

	if err := actionConfig.Init(restClientGetter, namespace, os.Getenv("HELM_DRIVER"), d.debug); err != nil {
		return err
	}

	if err := addDomain(path, d.Domain); err != nil {
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
		installer.IncludeCRDs = false
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

func (d *HelmDeployer) debug(format string, v ...interface{}) {
	if d.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

// releaseName returns chart name from path, without file extention and path, as release name:
// /tmp/a/b/b/ertia-core.tgz -> ertia-core
// /tmp/a/b/b/ertia-core -> ertia-core
func releaseName(path string) string {
	_, file := filepath.Split(path)
	ext := filepath.Ext(file)
	return strings.ToLower(file[0 : len(file)-len(ext)])
}

func addDomain(path, domain string) error {
	valuesFile := filepath.Join(path, helmValuesFile)

	content, err := ioutil.ReadFile(valuesFile)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(placeholderDomain)
	updatedContent := re.ReplaceAll(content, []byte(strings.TrimLeft(domain, ".")))

	if err := ioutil.WriteFile(valuesFile, updatedContent, 0644); err != nil {
		return err
	}

	return nil
}
