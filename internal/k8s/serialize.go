package k8s

import (
	"bufio"
	"bytes"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlDecoder "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	yamlEncoder "sigs.k8s.io/yaml"
)

func ParseYAMLFromString(yaml string) ([]K8sEntity, error) {
	buf := bytes.NewBuffer([]byte(yaml))
	return ParseYAML(buf)
}

// Parse the YAML into entities.
// Loosely based on
// https://github.com/kubernetes/cli-runtime/blob/d6a36215b15f83b94578f2ffce5d00447972e8ae/pkg/genericclioptions/resource/visitor.go#L583
func ParseYAML(k8sYaml io.Reader) ([]K8sEntity, error) {
	reader := bufio.NewReader(k8sYaml)
	decoder := yamlDecoder.NewYAMLOrJSONDecoder(reader, 4096)

	result := make([]K8sEntity, 0)
	for {
		ext := runtime.RawExtension{}
		if err := decoder.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		ext.Raw = bytes.TrimSpace(ext.Raw)

		// NOTE(nick): I LOL'd at the null check, but it's what kubectl does.
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}

		obj, _, err :=
			scheme.Codecs.UniversalDeserializer().Decode(ext.Raw, nil, nil)
		if err == nil {
			result = append(result, K8sEntity{
				Obj: obj,
			})
			continue
		}

		if !runtime.IsNotRegisteredError(err) {
			return nil, err
		}

		// If this is a NotRegisteredError, fallback to unstructured code
		obj, _, err =
			unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, nil)
		if err != nil {
			return nil, err
		}

		result = append(result, K8sEntity{
			Obj: obj,
		})
	}

	return result, nil
}

// Serializes the provided K8s object as YAML to the given writer.
//
// By convention, all K8s objects contain ObjectMetadata, Spec, and Status.
// This only serializes the metadata and spec, skipping the status.
func serializeSpec(obj runtime.Object, w io.Writer) error {
	json, err := specJSONIterator.Marshal(obj)
	if err != nil {
		return err
	}
	data, err := yamlEncoder.JSONToYAML(json)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// Serializes the provided K8s objects as YAML.
//
// By convention, all K8s objects contain ObjectMetadata, Spec, and Status.
// This only serializes the metadata and spec, skipping the status.
func SerializeSpecYAML(decoded []K8sEntity) (string, error) {
	buf := bytes.NewBuffer(nil)
	for i, obj := range decoded {
		if i != 0 {
			buf.Write([]byte("\n---\n"))
		}
		err := serializeSpec(obj.Obj, buf)
		if err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
