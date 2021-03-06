// Copyright 2018 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package main

import (
	"io"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
)

type aggOverloads struct {
	Agg       string
	Overloads []*oneArgOverload
}

// AggNameLower returns the aggregation name in lower case, e.g. "min".
func (a aggOverloads) AggNameLower() string {
	return strings.ToLower(a.Agg)
}

// AggNameTitle returns the aggregation name in title case, e.g. "Min".
func (a aggOverloads) AggNameTitle() string {
	return strings.Title(a.AggNameLower())
}

// Avoid unused warning for functions which are only used in templates.
var _ = aggOverloads{}.AggNameLower()
var _ = aggOverloads{}.AggNameTitle()

const minMaxAggTmpl = "pkg/sql/colexec/min_max_agg_tmpl.go"

func genMinMaxAgg(wr io.Writer) error {
	t, err := ioutil.ReadFile(minMaxAggTmpl)
	if err != nil {
		return err
	}

	s := string(t)

	s = strings.ReplaceAll(s, "_AGG_TITLE", "{{.AggNameTitle}}")
	s = strings.ReplaceAll(s, "_AGG", "{{$agg}}")
	s = strings.ReplaceAll(s, "_CANONICAL_TYPE_FAMILY", "{{.CanonicalTypeFamilyStr}}")
	s = strings.ReplaceAll(s, "_TYPE_WIDTH", typeWidthReplacement)
	s = strings.ReplaceAll(s, "_GOTYPESLICE", "{{.GoTypeSliceName}}")
	s = strings.ReplaceAll(s, "_GOTYPE", "{{.GoType}}")
	s = strings.ReplaceAll(s, "_TYPE", "{{.VecMethod}}")
	s = strings.ReplaceAll(s, "TemplateType", "{{.VecMethod}}")

	assignCmpRe := makeFunctionRegex("_ASSIGN_CMP", 3)
	s = assignCmpRe.ReplaceAllString(s, makeTemplateFunctionCall("Assign", 3))

	accumulateMinMax := makeFunctionRegex("_ACCUMULATE_MINMAX", 4)
	s = accumulateMinMax.ReplaceAllString(s, `{{template "accumulateMinMax" buildDict "Global" . "HasNulls" $4}}`)

	s = replaceManipulationFuncs(s)

	tmpl, err := template.New("min_max_agg").Funcs(template.FuncMap{"buildDict": buildDict}).Parse(s)

	if err != nil {
		return err
	}
	data := []aggOverloads{
		{
			Agg:       "MIN",
			Overloads: sameTypeComparisonOpToOverloads[tree.LT],
		},
		{
			Agg:       "MAX",
			Overloads: sameTypeComparisonOpToOverloads[tree.GT],
		},
	}
	return tmpl.Execute(wr, data)
}

func init() {
	registerGenerator(genMinMaxAgg, "min_max_agg.eg.go", minMaxAggTmpl)
}
