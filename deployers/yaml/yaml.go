package yaml

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
)

type YamlDeployer struct {
	KubeConfig string
	RestConfig *restclient.Config
	KubeClient *kubernetes.Clientset
}

func NewYamlDeployer(kubeCfgPath string) (*YamlDeployer, error) {

	restConfig, err := clientcmd.BuildConfigFromFlags(
		"", kubeCfgPath,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"Unable to load kubeconfig from %s: %v",
			kubeCfgPath, err,
		)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	return &YamlDeployer{
		KubeConfig: kubeCfgPath,
		RestConfig: restConfig,
		KubeClient: clientset,
	}, nil

}

func (d *YamlDeployer) Name() string {
	return "YamlDeployer"
}

func (d *YamlDeployer) CreateNamespace(ctx context.Context, namespace string) error {
	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}

	_, err := d.KubeClient.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{})
	return err
}

func (d *YamlDeployer) Deploy(ctx context.Context, namespace string, r io.Reader) error {

	cfg := d.RestConfig

	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 2. Prepare the dynamic client
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	// 3. Decode YAML manifest into unstructured.Unstructured, for each part of file...

	parts, err := GetDeploymentParts(r)

	for _, partialDeployment := range parts {

		if len(partialDeployment) < 5 {
			continue
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := GetDeployment(partialDeployment, obj)

		if gvk == nil {
			err = nil
			continue //Probably comment block, ignore
		}

		if err != nil {
			return err
		}

		if namespace != "" {
			obj.SetNamespace(namespace)
		}

		// 4. Find GVR
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		// 5. Obtain REST interface for the GVR
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			// namespaced resources should specify the namespace)
			dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			// for cluster-wide resources
			dr = dyn.Resource(mapping.Resource)
		}

		// 6. Marshal object into JSON
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		// 7. Create or Update the object with SSA
		//     types.ApplyPatchType indicates SSA.
		//     FieldManager specifies the field owner ID.
		_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "sample-controller",
		})

	}

	return err
}

func (d *YamlDeployer) DeployPath(ctx context.Context, namespace string, path string) error {

	r, err := os.Open(path)
	if err != nil {
		return err
	}

	cfg := d.RestConfig

	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 2. Prepare the dynamic client
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	// 3. Decode YAML manifest into unstructured.Unstructured, for each part of file...

	parts, err := GetDeploymentParts(r)

	for _, partialDeployment := range parts {

		if len(partialDeployment) < 5 {
			continue
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := GetDeployment(partialDeployment, obj)

		if gvk == nil {
			err = nil
			continue //Probably comment block, ignore
		}

		if err != nil {
			return err
		}

		if namespace != "" {
			obj.SetNamespace(namespace)
		}

		// 4. Find GVR
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		// 5. Obtain REST interface for the GVR
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			// namespaced resources should specify the namespace)
			dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			// for cluster-wide resources
			dr = dyn.Resource(mapping.Resource)
		}

		// 6. Marshal object into JSON
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		// 7. Create or Update the object with SSA
		//     types.ApplyPatchType indicates SSA.
		//     FieldManager specifies the field owner ID.
		_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
			FieldManager: "sample-controller",
		})

	}

	return err
}

func GetDeployment(deployment string, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	return decode([]byte(deployment), nil, into)
}

func GetDeploymentParts(r io.Reader) ([]string, error) {

	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(string(bytes), "---")

	return parts, nil
}
