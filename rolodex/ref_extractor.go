// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rolodex

import (
	"fmt"
	"regexp"
	"strings"
)

var refRegex = regexp.MustCompile(`['"]?\$ref['"]?\s*:\s*['"]?([^'"]*?)['"]`)

type RefType int

const (
	Local RefType = iota
	File
	HTTP
)

type ExtractedRef struct {
	Location string
	Type     RefType
}

func (r *ExtractedRef) GetFile() string {
	switch r.Type {
	case File, HTTP:
		location := strings.Split(r.Location, "#/")
		return location[0]
	default:
		return r.Location
	}
}

func (r *ExtractedRef) GetReference() string {
	switch r.Type {
	case File, HTTP:
		location := strings.Split(r.Location, "#/")
		return fmt.Sprintf("#/%s", location[1])
	default:
		return r.Location
	}
}

func ExtractRefValues(ref string) (location, id string) {
	split := strings.Split(ref, "#/")
	if len(split) > 1 && split[0] != "" {
		location = split[0]
		id = split[1]
	}
	if len(split) > 1 && split[0] == "" {
		id = split[1]
	}
	if len(split) == 1 {
		location = ref
	}
	return
}

func ExtractRefType(ref string) RefType {
	if strings.HasPrefix(ref, "http") {
		return HTTP
	}
	if strings.HasPrefix(ref, "/") {
		return File
	}
	if strings.HasPrefix(ref, "..") {
		return File
	}
	if strings.HasPrefix(ref, "./") {
		return File
	}
	split := strings.Split(ref, "#/")
	if len(split) > 1 && split[0] != "" {
		return File
	}
	if strings.HasSuffix(ref, ".yaml") {
		return File
	}
	if strings.HasSuffix(ref, ".json") {
		return File
	}
	return Local
}

func ExtractRefs(content string) [][]string {

	res := refRegex.FindAllStringSubmatch(content, -1)

	var results []*ExtractedRef
	for _, r := range res {
		results = append(results, &ExtractedRef{Location: r[1], Type: ExtractRefType(r[1])})
	}

	return res
}
