// SPDX-FileCopyrightText: 2023 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"testing"

	"github.com/crossplane/upjet/pkg/types/conversion/tfjson"
	"github.com/google/go-cmp/cmp"
)

func TestDeleteOmittedFields(t *testing.T) {
	type args struct {
		sch           map[string]*tfjson.Schema
		omittedFields []string
	}
	type want struct {
		sch map[string]*tfjson.Schema
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"No-op": {
			reason: "Should not make any changes if fields are not found.",
			args: args{
				sch: map[string]*tfjson.Schema{
					"top_level_a": {},
					"top_level_b": {},
				},
				omittedFields: []string{
					"top_level_c",
				},
			},
			want: want{
				sch: map[string]*tfjson.Schema{
					"top_level_a": {},
					"top_level_b": {},
				},
			},
		},
		"OmitTopLevelFields": {
			reason: "Should be able to omit top level fields.",
			args: args{
				sch: map[string]*tfjson.Schema{
					"top_level_a": {},
					"top_level_b": {},
				},
				omittedFields: []string{
					"top_level_a",
				},
			},
			want: want{
				sch: map[string]*tfjson.Schema{
					"top_level_b": {},
				},
			},
		},
		"OmitLeafNode": {
			reason: "Should be able to omit a leaf field.",
			args: args{
				sch: map[string]*tfjson.Schema{
					"top_level_a": {
						Elem: &tfjson.Resource{
							Schema: map[string]*tfjson.Schema{
								"down_one": {},
								"down_two": {},
							},
						},
					},
					"top_level_b": {
						Elem: &tfjson.Resource{
							Schema: map[string]*tfjson.Schema{
								"down_another": {},
							},
						},
					},
				},
				omittedFields: []string{
					"top_level_a.down_one",
				},
			},
			want: want{
				sch: map[string]*tfjson.Schema{
					"top_level_a": {
						Elem: &tfjson.Resource{
							Schema: map[string]*tfjson.Schema{
								"down_two": {},
							},
						},
					},
					"top_level_b": {
						Elem: &tfjson.Resource{
							Schema: map[string]*tfjson.Schema{
								"down_another": {},
							},
						},
					},
				},
			},
		},
		"OmitLeafNodeMultiple": {
			reason: "Should be able to omit multiple leaf fields.",
			args: args{
				sch: map[string]*tfjson.Schema{
					"top_level_a": {
						Elem: &tfjson.Resource{
							Schema: map[string]*tfjson.Schema{
								"down_one":        {},
								"down_one_prefix": {},
								"down_two":        {},
							},
						},
					},
					"top_level_b": {
						Elem: &tfjson.Resource{
							Schema: map[string]*tfjson.Schema{
								"down_another": {},
							},
						},
					},
				},
				omittedFields: []string{
					"top_level_a.down_one",
					"top_level_a.down_one_prefix",
				},
			},
			want: want{
				sch: map[string]*tfjson.Schema{
					"top_level_a": {
						Elem: &tfjson.Resource{
							Schema: map[string]*tfjson.Schema{
								"down_two": {},
							},
						},
					},
					"top_level_b": {
						Elem: &tfjson.Resource{
							Schema: map[string]*tfjson.Schema{
								"down_another": {},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			deleteOmittedFields(tc.args.sch, tc.args.omittedFields)
			if diff := cmp.Diff(tc.want.sch, tc.args.sch); diff != "" {
				t.Errorf("\n%s\ndeleteOmittedFields(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
