// SPDX-FileCopyrightText: 2023 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/crossplane/upjet/pkg/config/conversion"
	"github.com/crossplane/upjet/pkg/registry"
	"github.com/crossplane/upjet/pkg/types/conversion/tfjson"
)

func TestDefaultResource(t *testing.T) {
	identityConversion := conversion.NewIdentityConversionExpandPaths(conversion.AllVersions, conversion.AllVersions, nil)

	type args struct {
		name              string
		sch               *schema.Resource
		frameworkResource fwresource.Resource
		reg               *registry.Resource
		opts              []ResourceOption
	}

	cases := map[string]struct {
		reason string
		args   args
		want   *Resource
	}{
		"ThreeSectionsName": {
			reason: "It should return GVK properly for names with three sections",
			args: args{
				name: "aws_ec2_instance",
			},
			want: &Resource{
				Name:                           "aws_ec2_instance",
				ShortGroup:                     "ec2",
				Kind:                           "Instance",
				Version:                        "v1alpha1",
				ExternalName:                   NameAsIdentifier,
				References:                     map[string]Reference{},
				Sensitive:                      NopSensitive,
				UseAsync:                       true,
				SchemaElementOptions:           SchemaElementOptions{},
				ServerSideApplyMergeStrategies: ServerSideApplyMergeStrategies{},
				Conversions:                    []conversion.Conversion{identityConversion},
				OverrideFieldNames:             map[string]string{},
			},
		},
		"TwoSectionsName": {
			reason: "It should return GVK properly for names with three sections",
			args: args{
				name: "aws_instance",
			},
			want: &Resource{
				Name:                           "aws_instance",
				ShortGroup:                     "aws",
				Kind:                           "Instance",
				Version:                        "v1alpha1",
				ExternalName:                   NameAsIdentifier,
				References:                     map[string]Reference{},
				Sensitive:                      NopSensitive,
				UseAsync:                       true,
				SchemaElementOptions:           SchemaElementOptions{},
				ServerSideApplyMergeStrategies: ServerSideApplyMergeStrategies{},
				Conversions:                    []conversion.Conversion{identityConversion},
				OverrideFieldNames:             map[string]string{},
			},
		},
		"NameWithPrefixAcronym": {
			reason: "It should return prefix acronym in capital case",
			args: args{
				name: "aws_db_sql_server",
			},
			want: &Resource{
				Name:                           "aws_db_sql_server",
				ShortGroup:                     "db",
				Kind:                           "SQLServer",
				Version:                        "v1alpha1",
				ExternalName:                   NameAsIdentifier,
				References:                     map[string]Reference{},
				Sensitive:                      NopSensitive,
				UseAsync:                       true,
				SchemaElementOptions:           SchemaElementOptions{},
				ServerSideApplyMergeStrategies: ServerSideApplyMergeStrategies{},
				Conversions:                    []conversion.Conversion{identityConversion},
				OverrideFieldNames:             map[string]string{},
			},
		},
		"NameWithSuffixAcronym": {
			reason: "It should return suffix acronym in capital case",
			args: args{
				name: "aws_db_server_id",
			},
			want: &Resource{
				Name:                           "aws_db_server_id",
				ShortGroup:                     "db",
				Kind:                           "ServerID",
				Version:                        "v1alpha1",
				ExternalName:                   NameAsIdentifier,
				References:                     map[string]Reference{},
				Sensitive:                      NopSensitive,
				UseAsync:                       true,
				SchemaElementOptions:           SchemaElementOptions{},
				ServerSideApplyMergeStrategies: ServerSideApplyMergeStrategies{},
				Conversions:                    []conversion.Conversion{identityConversion},
				OverrideFieldNames:             map[string]string{},
			},
		},
		"NameWithMultipleAcronyms": {
			reason: "It should return both prefix & suffix acronyms in capital case",
			args: args{
				name: "aws_db_sql_server_id",
			},
			want: &Resource{
				Name:                           "aws_db_sql_server_id",
				ShortGroup:                     "db",
				Kind:                           "SQLServerID",
				Version:                        "v1alpha1",
				ExternalName:                   NameAsIdentifier,
				References:                     map[string]Reference{},
				Sensitive:                      NopSensitive,
				UseAsync:                       true,
				SchemaElementOptions:           SchemaElementOptions{},
				ServerSideApplyMergeStrategies: ServerSideApplyMergeStrategies{},
				Conversions:                    []conversion.Conversion{identityConversion},
				OverrideFieldNames:             map[string]string{},
			},
		},
	}

	// TODO(muvaf): Find a way to compare function pointers.
	ignoreUnexported := []cmp.Option{
		cmpopts.IgnoreFields(Sensitive{}, "fieldPaths", "AdditionalConnectionDetailsFn"),
		cmpopts.IgnoreFields(LateInitializer{}, "ignoredCanonicalFieldPaths", "conditionalIgnoredCanonicalFieldPaths"),
		cmpopts.IgnoreFields(ExternalName{}, "SetIdentifierArgumentFn", "GetExternalNameFn", "GetIDFn"),
		cmpopts.IgnoreUnexported(Resource{}),
		cmpopts.IgnoreUnexported(reflect.ValueOf(identityConversion).Elem().Interface()),
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := DefaultResource(tc.args.name, tc.args.sch, tc.args.frameworkResource, tc.args.reg, tc.args.opts...)
			if diff := cmp.Diff(tc.want, r, ignoreUnexported...); diff != "" {
				t.Errorf("\n%s\nDefaultResource(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestMoveToStatus(t *testing.T) {
	type args struct {
		sch    *tfjson.Resource
		fields []string
	}
	type want struct {
		sch *tfjson.Resource
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"DoesNotExist": {
			args: args{
				fields: []string{"topD"},
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {Type: tfjson.TypeString},
						"topC": {Type: tfjson.TypeString, Required: false},
					},
				},
			},
			want: want{
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {Type: tfjson.TypeString},
						"topC": {Type: tfjson.TypeString, Required: false},
					},
				},
			},
		},
		"TopLevelBasicFields": {
			args: args{
				fields: []string{"topA", "topB"},
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {Type: tfjson.TypeString},
						"topC": {Type: tfjson.TypeString, Required: false},
					},
				},
			},
			want: want{
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {
							Type:        tfjson.TypeString,
							Observation: true,
							Required:    true,
						},
						"topC": {
							Type:        tfjson.TypeString,
							Observation: true,
							Required:    true,
						},
					},
				},
			},
		},
		"ComplexFields": {
			args: args{
				fields: []string{"topA"},
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {
							Type: tfjson.TypeMap,
							Elem: &tfjson.Resource{
								Schema: map[string]*tfjson.Schema{
									"leafA": {
										Type: tfjson.TypeMap,
										Elem: &tfjson.Resource{
											Schema: map[string]*tfjson.Schema{
												"leafB": {
													Type:        tfjson.TypeString,
													Required:    false,
													Observation: false,
												},
												"leafC": {
													Type:        tfjson.TypeString,
													Required:    false,
													Observation: false,
												},
											},
										},
									},
								},
							},
						},
						"topB": {Type: tfjson.TypeString},
					},
				},
			},
			want: want{
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {
							Type:        tfjson.TypeMap,
							Observation: true,
							Required:    true,
							Elem: &tfjson.Resource{
								Schema: map[string]*tfjson.Schema{
									"leafA": {
										Type:        tfjson.TypeMap,
										Observation: true,
										Required:    true,
										Elem: &tfjson.Resource{
											Schema: map[string]*tfjson.Schema{
												"leafB": {
													Type:        tfjson.TypeString,
													Observation: true,
													Required:    true,
												},
												"leafC": {
													Type:        tfjson.TypeString,
													Observation: true,
													Required:    true,
												},
											},
										},
									},
								},
							},
						},
						"topB": {Type: tfjson.TypeString},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			MoveToStatus(tc.args.sch, tc.args.fields...)
			if diff := cmp.Diff(tc.want.sch, tc.args.sch); diff != "" {
				t.Errorf("\n%s\nMoveToStatus(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestMarkAsRequired(t *testing.T) {
	type args struct {
		sch    *tfjson.Resource
		fields []string
	}
	type want struct {
		sch *tfjson.Resource
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"DoesNotExist": {
			args: args{
				fields: []string{"topD"},
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {Type: tfjson.TypeString},
						"topC": {Type: tfjson.TypeString, Required: false},
					},
				},
			},
			want: want{
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {Type: tfjson.TypeString},
						"topC": {Type: tfjson.TypeString, Required: false},
					},
				},
			},
		},
		"TopLevelBasicFields": {
			args: args{
				fields: []string{"topB", "topC"},
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {Type: tfjson.TypeString},
						"topC": {Type: tfjson.TypeString, Required: false},
					},
				},
			},
			want: want{
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {Type: tfjson.TypeString},
						"topB": {
							Observation: false,
							Required:    true,
						},
						"topC": {
							Type:        tfjson.TypeString,
							Observation: false,
							Required:    true,
						},
					},
				},
			},
		},
		"ComplexFields": {
			args: args{
				fields: []string{"topA.leafA", "topA.leafA.leafC"},
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {
							Type: tfjson.TypeMap,
							Elem: &tfjson.Resource{
								Schema: map[string]*tfjson.Schema{
									"leafA": {
										Type: tfjson.TypeMap,
										Elem: &tfjson.Resource{
											Schema: map[string]*tfjson.Schema{
												"leafB": {Type: tfjson.TypeString},
												"leafC": {Type: tfjson.TypeString},
											},
										},
									},
								},
							},
						},
						"topB": {Type: tfjson.TypeString},
					},
				},
			},
			want: want{
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {
							Type: tfjson.TypeMap,
							Elem: &tfjson.Resource{
								Schema: map[string]*tfjson.Schema{
									"leafA": {
										Type:        tfjson.TypeMap,
										Observation: false,
										Required:    true,
										Elem: &tfjson.Resource{
											Schema: map[string]*tfjson.Schema{
												"leafB": {Type: tfjson.TypeString},
												"leafC": {
													Type:        tfjson.TypeString,
													Observation: false,
													Required:    true,
												},
											},
										},
									},
								},
							},
						},
						"topB": {Type: tfjson.TypeString},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			MarkAsRequired(tc.args.sch, tc.args.fields...)
			if diff := cmp.Diff(tc.want.sch, tc.args.sch); diff != "" {
				t.Errorf("\n%s\nMarkAsRequired(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestGetSchema(t *testing.T) {
	type args struct {
		sch       *tfjson.Resource
		fieldpath string
	}
	type want struct {
		sch *tfjson.Schema
	}
	schLeaf := &tfjson.Schema{
		Type: tfjson.TypeString,
	}
	schA := &tfjson.Schema{
		Type: tfjson.TypeMap,
		Elem: &tfjson.Resource{
			Schema: map[string]*tfjson.Schema{
				"fieldA": schLeaf,
			},
		},
	}
	res := &tfjson.Resource{
		Schema: map[string]*tfjson.Schema{
			"topA": schA,
		},
	}
	cases := map[string]struct {
		reason string
		args
		want
	}{
		"TopLevelField": {
			args: args{
				fieldpath: "topA",
				sch:       res,
			},
			want: want{
				sch: schA,
			},
		},
		"LeafField": {
			args: args{
				fieldpath: "topA.fieldA",
				sch:       res,
			},
			want: want{
				sch: schLeaf,
			},
		},
		"TopLevelFieldNotFound": {
			args: args{
				fieldpath: "topB",
				sch:       res,
			},
			want: want{
				sch: nil,
			},
		},
		"LeafFieldNotFound": {
			args: args{
				fieldpath: "topA.olala.omama",
				sch:       res,
			},
			want: want{
				sch: nil,
			},
		},
		"TopFieldIsNotMap": {
			args: args{
				fieldpath: "topA.topB",
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {Type: tfjson.TypeString},
					},
				},
			},
			want: want{
				sch: nil,
			},
		},
		"MiddleFieldIsNotResource": {
			args: args{
				fieldpath: "topA.topB.topC",
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {
							Elem: &tfjson.Resource{
								Schema: map[string]*tfjson.Schema{
									"topB": {
										Elem: &tfjson.Schema{},
									},
								},
							},
						},
					},
				},
			},
			want: want{
				sch: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sch := GetSchema(tc.args.sch, tc.args.fieldpath)
			if diff := cmp.Diff(tc.want.sch, sch); diff != "" {
				t.Errorf("\n%s\nGetSchema(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestManipulateAllFieldsInSchema(t *testing.T) {
	type args struct {
		sch *tfjson.Resource
		op  func(sch *tfjson.Schema)
	}
	type want struct {
		sch *tfjson.Resource
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"SetEmptyDescription": {
			args: args{
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {
							Description: "topADescription",
							Type:        tfjson.TypeMap,
							Elem: &tfjson.Resource{
								Schema: map[string]*tfjson.Schema{
									"leafA": {
										Description: "leafADescription",
										Type:        tfjson.TypeMap,
										Elem: &tfjson.Resource{
											Schema: map[string]*tfjson.Schema{
												"leafB": {
													Description: "",
													Type:        tfjson.TypeString,
												},
												"leafC": {
													Description: "leafCDescription",
													Type:        tfjson.TypeString,
												},
											},
										},
									},
								},
							},
						},
						"topB": {Type: tfjson.TypeString},
					},
				},
				op: func(sch *tfjson.Schema) {
					sch.Description = ""
				},
			},
			want: want{
				sch: &tfjson.Resource{
					Schema: map[string]*tfjson.Schema{
						"topA": {
							Description: "",
							Type:        tfjson.TypeMap,
							Elem: &tfjson.Resource{
								Schema: map[string]*tfjson.Schema{
									"leafA": {
										Description: "",
										Type:        tfjson.TypeMap,
										Elem: &tfjson.Resource{
											Schema: map[string]*tfjson.Schema{
												"leafB": {
													Description: "",
													Type:        tfjson.TypeString,
												},
												"leafC": {
													Description: "",
													Type:        tfjson.TypeString,
												},
											},
										},
									},
								},
							},
						},
						"topB": {Type: tfjson.TypeString, Description: ""},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ManipulateEveryField(tc.args.sch, tc.args.op)
			if diff := cmp.Diff(tc.want.sch, tc.args.sch); diff != "" {
				t.Errorf("\n%s\nMoveToStatus(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
