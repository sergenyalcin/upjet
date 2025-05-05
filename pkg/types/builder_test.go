// SPDX-FileCopyrightText: 2023 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"go/token"
	"go/types"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/crossplane/upjet/pkg/types/conversion/tfjson"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/crossplane/upjet/pkg/config"
)

func TestBuilder_generateTypeName(t *testing.T) {
	type args struct {
		existing []string
		suffix   string
		names    []string

		overrideFieldNames map[string]string
	}
	type want struct {
		out string
		err error
	}
	cases := map[string]struct {
		args
		want
	}{
		"NoExisting": {
			args: args{
				existing: []string{
					"SomeOtherType",
				},
				suffix: "Parameters",
				names: []string{
					"Subnetwork",
				},
			},
			want: want{
				out: "SubnetworkParameters",
				err: nil,
			},
		},
		"NoExistingMultipleIndexes": {
			args: args{
				existing: []string{
					"SomeOtherType",
				},
				suffix: "Parameters",
				names: []string{
					"RouterNat",
					"Subnetwork",
				},
			},
			want: want{
				out: "SubnetworkParameters",
				err: nil,
			},
		},
		"NoIndexExists": {
			args: args{
				existing: []string{
					"SubnetworkParameters",
				},
				suffix: "Parameters",
				names: []string{
					"Subnetwork",
				},
			},
			want: want{
				out: "SubnetworkParameters_2",
				err: nil,
			},
		},
		"MultipleIndexesExist": {
			args: args{
				existing: []string{
					"SubnetworkParameters",
					"SubnetworkParameters_2",
					"SubnetworkParameters_3",
					"SubnetworkParameters_4",
				},
				suffix: "Parameters",
				names: []string{
					"Subnetwork",
				},
			},
			want: want{
				out: "SubnetworkParameters_5",
				err: nil,
			},
		},
		"ErrIfAllIndexesExist": {
			args: args{
				existing: []string{
					"SubnetworkParameters",
					"SubnetworkParameters_2",
					"SubnetworkParameters_3",
					"SubnetworkParameters_4",
					"SubnetworkParameters_5",
					"SubnetworkParameters_6",
					"SubnetworkParameters_7",
					"SubnetworkParameters_8",
					"SubnetworkParameters_9",
				},
				suffix: "Parameters",
				names: []string{
					"Subnetwork",
				},
			},
			want: want{
				err: errors.Errorf("could not generate a unique name for %s", "SubnetworkParameters"),
			},
		},
		"MultipleNamesPrependsBeforeIndexing": {
			args: args{
				existing: []string{
					"SubnetworkParameters",
				},
				suffix: "Parameters",
				names: []string{
					"RouterNat",
					"Subnetwork",
				},
			},
			want: want{
				out: "RouterNatSubnetworkParameters",
				err: nil,
			},
		},
		"MultipleNamesUsesIndexingIfNeeded": {
			args: args{
				existing: []string{
					"SubnetworkParameters",
					"RouterNatSubnetworkParameters",
				},
				suffix: "Parameters",
				names: []string{
					"RouterNat",
					"Subnetwork",
				},
			},
			want: want{
				out: "RouterNatSubnetworkParameters_2",
				err: nil,
			},
		},
		"AnySuffixWouldWorkSame": {
			args: args{
				existing: []string{
					"SubnetworkObservation",
					"SubnetworkObservation_2",
					"SubnetworkObservation_3",
					"SubnetworkObservation_4",
				},
				suffix: "Observation",
				names: []string{
					"Subnetwork",
				},
			},
			want: want{
				out: "SubnetworkObservation_5",
				err: nil,
			},
		},
		"OverrideFieldNames": {
			args: args{
				suffix: "Parameters",
				names: []string{
					"Cluster",
					"Tag",
				},
				overrideFieldNames: map[string]string{
					"TagParameters": "ClusterTagParameters",
				},
			},
			want: want{
				out: "ClusterTagParameters",
				err: nil,
			},
		},
	}
	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			p := types.NewPackage("path/to/test", "test")
			for _, s := range tc.existing {
				p.Scope().Insert(types.NewTypeName(token.NoPos, p, s, &types.Struct{}))
			}

			g := &Builder{
				Package: p,
			}
			got, gotErr := generateTypeName(tc.args.suffix, g.Package, tc.args.overrideFieldNames, tc.args.names...)
			if diff := cmp.Diff(tc.want.err, gotErr, test.EquateErrors()); diff != "" {
				t.Fatalf("generateTypeName(...): -want error, +got error: %s", diff)
			}
			if diff := cmp.Diff(tc.want.out, got); diff != "" {
				t.Errorf("generateTypeName(...) out = %v, want %v", got, tc.want.out)
			}
		})
	}
}

func TestBuild(t *testing.T) {
	type args struct {
		cfg *config.Resource
	}
	type want struct {
		forProvider     string
		atProvider      string
		validationRules string
		err             error
	}
	cases := map[string]struct {
		args
		want
	}{
		"Base_Types": {
			args: args{
				cfg: &config.Resource{
					UpjetResource: &tfjson.Resource{
						Schema: map[string]*tfjson.Schema{
							"name": {
								Type:     tfjson.TypeString,
								Required: true,
							},
							"id": {
								Type:     tfjson.TypeString,
								Required: true,
							},
							"enable": {
								Type:     tfjson.TypeBool,
								Required: false,
							},
							"value": {
								Type:        tfjson.TypeFloat,
								Observation: true,
								Required:    true,
							},
							"config": {
								Type:        tfjson.TypeString,
								Observation: true,
								Required:    true,
							},
						},
					},
				},
			},
			want: want{
				forProvider: `type example.Parameters struct{Enable *bool "json:\"enable,omitempty\" tf:\"enable,omitempty\""; ID *string "json:\"id,omitempty\" tf:\"id,omitempty\""; Name *string "json:\"name,omitempty\" tf:\"name,omitempty\""}`,
				atProvider:  `type example.Observation struct{Config *string "json:\"config,omitempty\" tf:\"config,omitempty\""; Enable *bool "json:\"enable,omitempty\" tf:\"enable,omitempty\""; ID *string "json:\"id,omitempty\" tf:\"id,omitempty\""; Name *string "json:\"name,omitempty\" tf:\"name,omitempty\""; Value *float64 "json:\"value,omitempty\" tf:\"value,omitempty\""}`,
				validationRules: `
// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.id) || (has(self.initProvider) && has(self.initProvider.id))",message="spec.forProvider.id is a required parameter"
// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.name) || (has(self.initProvider) && has(self.initProvider.name))",message="spec.forProvider.name is a required parameter"`,
			},
		},
		"Resource_Types": {
			args: args{
				cfg: &config.Resource{
					UpjetResource: &tfjson.Resource{
						Schema: map[string]*tfjson.Schema{
							"list": {
								Type:     tfjson.TypeList,
								Required: true,
								Elem: &tfjson.Schema{
									Type:     tfjson.TypeString,
									Required: true,
								},
							},
							"resource_in": {
								Type:     tfjson.TypeMap,
								Required: true,
								Elem:     &tfjson.Resource{},
							},
							"resource_out": {
								Type:        tfjson.TypeMap,
								Observation: true,
								Required:    false,
								Elem:        &tfjson.Resource{},
							},
						},
					},
				},
			},
			want: want{
				forProvider: `type example.Parameters struct{List []*string "json:\"list,omitempty\" tf:\"list,omitempty\""; ResourceIn map[string]example.ResourceInParameters "json:\"resourceIn,omitempty\" tf:\"resource_in,omitempty\""}`,
				atProvider:  `type example.Observation struct{List []*string "json:\"list,omitempty\" tf:\"list,omitempty\""; ResourceIn map[string]example.ResourceInParameters "json:\"resourceIn,omitempty\" tf:\"resource_in,omitempty\""; ResourceOut map[string]example.ResourceOutObservation "json:\"resourceOut,omitempty\" tf:\"resource_out,omitempty\""}`,
				validationRules: `
// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.list) || (has(self.initProvider) && has(self.initProvider.list))",message="spec.forProvider.list is a required parameter"
// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.resourceIn) || (has(self.initProvider) && has(self.initProvider.resourceIn))",message="spec.forProvider.resourceIn is a required parameter"`,
			},
		},
		"Sensitive_Fields": {
			args: args{
				cfg: &config.Resource{
					UpjetResource: &tfjson.Resource{
						Schema: map[string]*tfjson.Schema{
							"key_1": {
								Type:      tfjson.TypeString,
								Required:  false,
								Sensitive: true,
							},
							"key_2": {
								Type:      tfjson.TypeString,
								Sensitive: true,
							},
							"key_3": {
								Type:      tfjson.TypeList,
								Sensitive: true,
							},
						},
					},
				},
			},
			want: want{
				forProvider:     `type example.Parameters struct{Key1SecretRef *github.com/crossplane/crossplane-runtime/apis/common/v1.SecretKeySelector "json:\"key1SecretRef,omitempty\" tf:\"-\""; Key2SecretRef *github.com/crossplane/crossplane-runtime/apis/common/v1.SecretKeySelector "json:\"key2SecretRef,omitempty\" tf:\"-\""; Key3SecretRef *[]github.com/crossplane/crossplane-runtime/apis/common/v1.SecretKeySelector "json:\"key3SecretRef,omitempty\" tf:\"-\""}`,
				atProvider:      `type example.Observation struct{}`,
				validationRules: "",
			},
		},
		"Invalid_Sensitive_Fields": {
			args: args{
				cfg: &config.Resource{
					Name: "test_resource",
					UpjetResource: &tfjson.Resource{
						Schema: map[string]*tfjson.Schema{
							"key_1": {
								Type:      tfjson.TypeFloat,
								Sensitive: true,
							},
						},
					},
				},
			},
			want: want{
				err: errors.Wrapf(fmt.Errorf(`got type %q for field %q, only types "string", "*string", []string, []*string, "map[string]string" and "map[string]*string" supported as sensitive`, "*float64", "Key1"), `cannot build the Types for resource "test_resource"`),
			},
		},
		"References": {
			args: args{
				cfg: &config.Resource{
					UpjetResource: &tfjson.Resource{
						Schema: map[string]*tfjson.Schema{
							"name": {
								Type:     tfjson.TypeString,
								Required: true,
							},
							"reference_id": {
								Type:     tfjson.TypeString,
								Required: true,
							},
						},
					},
					References: map[string]config.Reference{
						"reference_id": {
							Type:         "string",
							RefFieldName: "ExternalResourceID",
						},
					},
				},
			},
			want: want{
				forProvider: `type example.Parameters struct{Name *string "json:\"name,omitempty\" tf:\"name,omitempty\""; ReferenceID *string "json:\"referenceId,omitempty\" tf:\"reference_id,omitempty\""; ExternalResourceID *github.com/crossplane/crossplane-runtime/apis/common/v1.Reference "json:\"externalResourceId,omitempty\" tf:\"-\""; ReferenceIDSelector *github.com/crossplane/crossplane-runtime/apis/common/v1.Selector "json:\"referenceIdSelector,omitempty\" tf:\"-\""}`,
				atProvider:  `type example.Observation struct{Name *string "json:\"name,omitempty\" tf:\"name,omitempty\""; ReferenceID *string "json:\"referenceId,omitempty\" tf:\"reference_id,omitempty\""}`,
				validationRules: `
// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.name) || (has(self.initProvider) && has(self.initProvider.name))",message="spec.forProvider.name is a required parameter"`,
			},
		},
		"Invalid_Schema_Type": {
			args: args{
				cfg: &config.Resource{
					Name: "test_resource",
					UpjetResource: &tfjson.Resource{
						Schema: map[string]*tfjson.Schema{
							"name": {
								Type:     tfjson.TypeInvalid,
								Required: true,
							},
						},
					},
				},
			},
			want: want{
				err: errors.Wrapf(errors.Wrapf(errors.Errorf("invalid schema type %s", "invalid"), "cannot infer type from schema of field %s", "name"), `cannot build the Types for resource "test_resource"`),
			},
		},
		"Validation_Rules_With_Keywords": {
			args: args{
				cfg: &config.Resource{
					UpjetResource: &tfjson.Resource{
						Schema: map[string]*tfjson.Schema{
							"name": {
								Type:     tfjson.TypeString,
								Required: true,
							},
							// "namespace" is a cel reserved value and should be wrapped when used in
							// validation rules (i.e., __namespace__)
							"namespace": {
								Type:     tfjson.TypeString,
								Required: true,
							},
						},
					},
				},
			},
			want: want{
				forProvider: `type example.Parameters struct{Name *string "json:\"name,omitempty\" tf:\"name,omitempty\""; Namespace *string "json:\"namespace,omitempty\" tf:\"namespace,omitempty\""}`,
				atProvider:  `type example.Observation struct{Name *string "json:\"name,omitempty\" tf:\"name,omitempty\""; Namespace *string "json:\"namespace,omitempty\" tf:\"namespace,omitempty\""}`,
				validationRules: `
// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.name) || (has(self.initProvider) && has(self.initProvider.name))",message="spec.forProvider.name is a required parameter"
// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.__namespace__) || (has(self.initProvider) && has(self.initProvider.__namespace__))",message="spec.forProvider.namespace is a required parameter"`,
			},
		},
	}
	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			builder := NewBuilder(types.NewPackage("example", ""))
			g, err := builder.Build(tc.cfg)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Fatalf("Build(...): -want error, +got error: %s", diff)
			}
			if g.ForProviderType != nil {
				if diff := cmp.Diff(tc.want.forProvider, g.ForProviderType.Obj().String()); diff != "" {
					t.Fatalf("Build(...): -want forProvider, +got forProvider: %s", diff)
				}
			}
			if g.AtProviderType != nil {
				if diff := cmp.Diff(tc.want.atProvider, g.AtProviderType.Obj().String()); diff != "" {
					t.Fatalf("Build(...): -want atProvider, +got atProvider: %s", diff)
				}
			}
			if diff := cmp.Diff(tc.want.validationRules, g.ValidationRules); diff != "" {
				t.Fatalf("Build(...): -want validationRules, +got validationRules: %s", diff)
			}
		})
	}
}
