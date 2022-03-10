package yaml

import (
	"context"
	"fmt"
	"io/ioutil"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type YamlDeployer struct {
	KubeConfig string
	RestConfig *restclient.Config
	KubeClient *kubernetes.Clientset
}

func NewYamlDeployer(kubeCfgPath string) (*YamlDeployer, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeCfgPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to load kubeconfig from %s: %v", kubeCfgPath, err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	return &YamlDeployer{
		KubeConfig: kubeCfgPath,
		KubeClient: clientset,
		RestConfig: restConfig,
	}, nil
}

func (d *YamlDeployer) Name() string {
	return "YamlDeployer"
}

func (d *YamlDeployer) CreateNamespace(ctx context.Context, namespace string) error {
	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}

	_, err := d.KubeClient.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (d *YamlDeployer) DeployPath(ctx context.Context, namespace string, path string) error {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = forEachObjectInYAML(ctx, f, namespace, deploy, d.RestConfig)
	if err != nil {
		return fmt.Errorf("deploy failed: %s", err)
	}

	return nil
}
