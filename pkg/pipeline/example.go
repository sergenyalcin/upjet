/*
Copyright 2022 Upbound Inc.
*/

package pipeline

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	xpmeta "github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/upbound/upjet/pkg/config"
	tjtypes "github.com/upbound/upjet/pkg/types"
)

var (
	reRef  = regexp.MustCompile(`\${(.+)}`)
	reFile = regexp.MustCompile(`file\("(.+)"\)`)
)

type pavedWithManifest struct {
	manifestPath string
	paved        *fieldpath.Paved
	refsResolved bool
}

// ExampleGenerator represents a pipeline for generating example manifests.
// Generates example manifests for Terraform resources under examples-generated.
type ExampleGenerator struct {
	rootDir        string
	configResource map[string]*config.Resource
	resources      map[string]*pavedWithManifest
}

// NewExampleGenerator returns a configured ExampleGenerator
func NewExampleGenerator(rootDir string, configResource map[string]*config.Resource) *ExampleGenerator {
	return &ExampleGenerator{
		rootDir:        rootDir,
		configResource: configResource,
		resources:      make(map[string]*pavedWithManifest),
	}
}

// StoreExamples stores the generated example manifests under examples-generated in
// their respective API groups.
func (eg *ExampleGenerator) StoreExamples() error {
	for n, pm := range eg.resources {
		if err := eg.resolveReferencesOfPaved(pm); err != nil {
			return errors.Wrapf(err, "cannot resolve references for resource: %s", n)
		}
		u := pm.paved.UnstructuredContent()
		delete(u["spec"].(map[string]interface{})["forProvider"].(map[string]interface{}), "depends_on")
		buff, err := yaml.Marshal(u)
		if err != nil {
			return errors.Wrapf(err, "cannot marshal example manifest for resource: %s", n)
		}
		manifestDir := filepath.Dir(pm.manifestPath)
		if err := os.MkdirAll(manifestDir, 0750); err != nil {
			return errors.Wrapf(err, "cannot mkdir %s", manifestDir)
		}
		// no sensitive info in the example manifest
		if err := ioutil.WriteFile(pm.manifestPath, buff, 0644); err != nil { // nolint:gosec
			return errors.Wrapf(err, "cannot write example manifest file %s for resource %s", pm.manifestPath, n)
		}
	}
	return nil
}

func (eg *ExampleGenerator) resolveReferencesOfPaved(pm *pavedWithManifest) error {
	if pm.refsResolved {
		return nil
	}
	pm.refsResolved = true
	return errors.Wrap(eg.resolveReferences(pm.paved.UnstructuredContent()), "failed to resolve references of paved")
}

func (eg *ExampleGenerator) resolveReferences(params map[string]interface{}) error { // nolint:gocyclo
	for k, v := range params {
		switch t := v.(type) {
		case map[string]interface{}:
			if err := eg.resolveReferences(t); err != nil {
				return err
			}

		case []interface{}:
			for _, e := range t {
				eM, ok := e.(map[string]interface{})
				if !ok {
					continue
				}
				if err := eg.resolveReferences(eM); err != nil {
					return err
				}
			}

		case string:
			g := reRef.FindStringSubmatch(t)
			if len(g) != 2 {
				continue
			}
			path := strings.Split(g[1], ".")
			// expected reference format is <resource type>.<resource name>.<field name>
			if len(path) < 3 {
				continue
			}
			pm := eg.resources[path[0]]
			if pm == nil || pm.paved == nil {
				continue
			}
			if err := eg.resolveReferencesOfPaved(pm); err != nil {
				return errors.Wrapf(err, "cannot recursively resolve references for %q", path[0])
			}
			pathStr := strings.Join(append([]string{"spec", "forProvider"}, path[2:]...), ".")
			s, err := pm.paved.GetString(pathStr)
			if fieldpath.IsNotFound(err) {
				continue
			}
			if err != nil {
				return errors.Wrapf(err, "cannot get string value from paved: %s", pathStr)
			}
			params[k] = s
		}
	}
	return nil
}

// Generate generates an example manifest for the specified Terraform resource.
func (eg *ExampleGenerator) Generate(group, version string, r *config.Resource, fieldTransformations map[string]tjtypes.Transformation) error {
	rm := eg.configResource[r.Name].MetaResource
	if rm == nil || len(rm.Examples) == 0 {
		return nil
	}
	exampleParams := rm.Examples[0].Paved.UnstructuredContent()
	transformFields(exampleParams, r.ExternalName.OmittedFields, fieldTransformations, "")

	metadata := map[string]interface{}{
		"name": "example",
	}
	if len(rm.ExternalName) != 0 {
		metadata["annotations"] = map[string]string{
			xpmeta.AnnotationKeyExternalName: rm.ExternalName,
		}
	}
	example := map[string]interface{}{
		"apiVersion": fmt.Sprintf("%s/%s", group, version),
		"kind":       r.Kind,
		"metadata":   metadata,
		"spec": map[string]interface{}{
			"forProvider": exampleParams,
		},
	}
	manifestDir := filepath.Join(eg.rootDir, "examples-generated", strings.ToLower(strings.Split(group, ".")[0]))
	eg.resources[r.Name] = &pavedWithManifest{
		manifestPath: filepath.Join(manifestDir, fmt.Sprintf("%s.yaml", strings.ToLower(r.Kind))),
		paved:        fieldpath.Pave(example),
	}
	return nil
}

func getHierarchicalName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return fmt.Sprintf("%s.%s", prefix, name)
}

func transformFields(params map[string]interface{}, omittedFields []string, t map[string]tjtypes.Transformation, namePrefix string) { // nolint:gocyclo
	for _, hn := range omittedFields {
		for n := range params {
			if hn == getHierarchicalName(namePrefix, n) {
				delete(params, n)
				break
			}
		}
	}

	for n, v := range params {
		switch pT := v.(type) {
		case map[string]interface{}:
			transformFields(pT, omittedFields, t, getHierarchicalName(namePrefix, n))

		case []interface{}:
			for _, e := range pT {
				eM, ok := e.(map[string]interface{})
				if !ok {
					continue
				}
				transformFields(eM, omittedFields, t, getHierarchicalName(namePrefix, n))
			}
		}
	}

	for hn, transform := range t {
		for n, v := range params {
			if hn == getHierarchicalName(namePrefix, n) {
				delete(params, n)
				if transform.IsRef {
					if !transform.IsSensitive {
						params[transform.TransformedName] = getRefField(v,
							map[string]interface{}{
								"name": "example",
							})
					} else {
						secretName, secretKey := getSecretRef(v)
						params[transform.TransformedName] = getRefField(v,
							map[string]interface{}{
								"name":      secretName,
								"namespace": "crossplane-system",
								"key":       secretKey,
							})
					}
				} else {
					params[transform.TransformedName] = v
				}
				break
			}
		}
	}
}

func getRefField(v interface{}, ref map[string]interface{}) interface{} {
	switch v.(type) {
	case []interface{}:
		return []interface{}{
			ref,
		}

	default:
		return ref
	}
}

func getSecretRef(v interface{}) (string, string) {
	secretName := "example-secret"
	secretKey := "example-key"
	s, ok := v.(string)
	if !ok {
		return secretName, secretKey
	}
	g := reRef.FindStringSubmatch(s)
	if len(g) != 2 {
		return secretName, secretKey
	}
	f := reFile.FindStringSubmatch(g[1])
	switch {
	case len(f) == 2: // then a file reference
		_, file := filepath.Split(f[1])
		secretKey = fmt.Sprintf("attribute.%s", file)
	default:
		parts := strings.Split(g[1], ".")
		if len(parts) < 3 {
			return secretName, secretKey
		}
		secretName = fmt.Sprintf("example-%s", strings.Join(strings.Split(parts[0], "_")[1:], "-"))
		secretKey = fmt.Sprintf("attribute.%s", strings.Join(parts[2:], "."))
	}
	return secretName, secretKey
}
