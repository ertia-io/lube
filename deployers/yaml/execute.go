package yaml

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

const fieldOwnerID = "lube-ctl"

// forEachObjectInYAMLActionFunc is a function that is executed against each
// object found in a YAML document.
type forEachObjectInYAMLActionFunc func(
	context.Context,
	*unstructured.Unstructured,
	*restclient.Config,
) error

// forEachObjectInYAML executes actionFn for each object in the provided YAML.
// If an error is returned then no further objects are processed.
// The data may be a single YAML document or multidoc YAML.
// When a non-empty namespace is provided then all objects are assigned the
// the namespace prior to any other actions being performed with or to the
// object.
func forEachObjectInYAML(
	ctx context.Context,
	data []byte,
	namespace string,
	actionFn forEachObjectInYAMLActionFunc,
	config *restclient.Config,
) error {

	chanObj, chanErr := decodeToUnstructured(data)
	for {
		select {
		case obj := <-chanObj:
			if obj == nil {
				return nil
			}
			if namespace != "" {
				obj.SetNamespace(namespace)
			}
			if err := actionFn(ctx, obj, config); err != nil {
				return err
			}
		case err := <-chanErr:
			if err == nil {
				return nil
			}
			return fmt.Errorf("received error while decoding yaml: %s", err)
		}
	}
}

var deploy forEachObjectInYAMLActionFunc = func(
	ctx context.Context, obj *unstructured.Unstructured, config *restclient.Config,
) error {
	time.Sleep(500 * time.Millisecond)

	// Discovery client
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return err
	}

	// Dynamic client
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	gvk := obj.GroupVersionKind()

	// Find Group Version Resource (GVR)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// Obtain REST interface for the GVR
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// for cluster-wide resources
		dr = dyn.Resource(mapping.Resource)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// Create or Update the object with Server Side Apply (SSA)
	//     types.ApplyPatchType indicates SSA.
	//     FieldManager specifies the field owner ID.
	_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data,
		metav1.PatchOptions{FieldManager: fieldOwnerID})
	if err != nil {
		return err
	}

	return nil
}
