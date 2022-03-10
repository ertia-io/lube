package yaml

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// decodeToUnstructured unmarshals a YAML document or multidoc YAML as unstructured
// objects, placing each decoded object into a channel.
func decodeToUnstructured(data []byte) (<-chan *unstructured.Unstructured, <-chan error) {
	var (
		chanErr         = make(chan error)
		chanObj         = make(chan *unstructured.Unstructured)
		multidocReader  = utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
		decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	)

	go func() {
		defer close(chanErr)
		defer close(chanObj)

		// Iterate over the data until Read returns io.EOF. Every successful
		// read returns a complete YAML document.
		for {
			buf, err := multidocReader.Read()
			if err != nil {
				if err == io.EOF {
					return
				}
				chanErr <- fmt.Errorf("failed to read yaml data: %s", err)
				return
			}

			// Define the unstructured object into which the YAML document will be
			// unmarshaled.
			obj := &unstructured.Unstructured{}

			// Unmarshal the YAML document into the unstructured object.
			_, _, err = decUnstructured.Decode(buf, nil, obj)
			if err != nil {
				chanErr <- fmt.Errorf("failed to unmarshal yaml data: %s", err)
				return
			}

			// Do not use this YAML doc if it is unkind.
			if obj.GetKind() == "" {
				continue
			}

			// Place the unstructured object into the channel.
			chanObj <- obj
		}
	}()

	return chanObj, chanErr
}
