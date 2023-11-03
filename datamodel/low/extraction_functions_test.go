// Copyright 2022-2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package low

import (
	"context"
	"crypto/sha256"
	"fmt"
	"golang.org/x/sync/syncmap"
	"gopkg.in/yaml.v3"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
)

func TestFindItemInMap(t *testing.T) {
	v := make(map[KeyReference[string]]ValueReference[string])
	v[KeyReference[string]{
		Value: "pizza",
	}] = ValueReference[string]{
		Value: "pie",
	}
	assert.Equal(t, "pie", FindItemInMap("pizza", v).Value)
}

func TestFindItemInMap_WrongCase(t *testing.T) {
	v := make(map[KeyReference[string]]ValueReference[string])
	v[KeyReference[string]{
		Value: "pizza",
	}] = ValueReference[string]{
		Value: "pie",
	}
	assert.Equal(t, "pie", FindItemInMap("PIZZA", v).Value)
}

func TestFindItemInMap_Error(t *testing.T) {
	v := make(map[KeyReference[string]]ValueReference[string])
	v[KeyReference[string]{
		Value: "pizza",
	}] = ValueReference[string]{
		Value: "pie",
	}
	assert.Nil(t, FindItemInMap("nuggets", v))
}

func TestLocateRefNode(t *testing.T) {

	yml := `components:
  schemas:
    cake:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/cake'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, _, _ := LocateRefNode(cNode.Content[0], idx)
	assert.NotNil(t, located)

}

func TestLocateRefNode_BadNode(t *testing.T) {

	yml := `components:
  schemas:
    cake:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `yes: mate` // useless.

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, _, err := LocateRefNode(cNode.Content[0], idx)

	// should both be empty.
	assert.Nil(t, located)
	assert.Nil(t, err)

}

func TestLocateRefNode_Path(t *testing.T) {

	yml := `paths:
  /burger/time:
    description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/paths/~1burger~1time'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, _, _ := LocateRefNode(cNode.Content[0], idx)
	assert.NotNil(t, located)

}

func TestLocateRefNode_Path_NotFound(t *testing.T) {

	yml := `paths:
  /burger/time:
    description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/paths/~1burger~1time-somethingsomethingdarkside-somethingsomethingcomplete'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	located, _, err := LocateRefNode(cNode.Content[0], idx)
	assert.Nil(t, located)
	assert.Error(t, err)

}

type pizza struct {
	Description NodeReference[string]
}

func (p *pizza) Build(_ context.Context, _, _ *yaml.Node, _ *index.SpecIndex) error {
	return nil
}

func TestExtractObject(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `tags:
  description: hello pizza`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "hello pizza", tag.Value.Description.Value)
}

func TestExtractObject_Ref(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hello pizza`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "hello pizza", tag.Value.Description.Value)
}

func TestExtractObject_DoubleRef(t *testing.T) {

	yml := `components:
  schemas:
    cake:
      description: cake time!
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "cake time!", tag.Value.Description.Value)
}

func TestExtractObject_DoubleRef_Circular(t *testing.T) {
	yml := `components:
  schemas:
    loopy:
      $ref: '#/components/schemas/cake'
    cake:
      $ref: '#/components/schemas/loopy'
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// circular references are detected by the resolver, so lets run it!
	resolv := index.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `tags:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
	assert.Error(t, err)
	assert.Equal(t, "cake -> loopy -> cake", idx.GetCircularReferences()[0].GenerateJourneyPath())
}

func TestExtractObject_DoubleRef_Circular_Fail(t *testing.T) {
	yml := `components:
  schemas:
    loopy:
      $ref: '#/components/schemas/cake'
    cake:
      $ref: '#/components/schemas/loopy'
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// circular references are detected by the resolver, so lets run it!
	resolv := index.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `tags:
  $ref: #BORK`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza](context.Background(), "tags", &cNode, idx)
	assert.Error(t, err)
}

func TestExtractObject_DoubleRef_Circular_Direct(t *testing.T) {

	yml := `components:
  schemas:
    loopy:
      $ref: '#/components/schemas/cake'
    cake:
      $ref: '#/components/schemas/loopy'
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// circular references are detected by the resolver, so lets run it!
	resolv := index.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `$ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza](context.Background(), "tags", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Equal(t, "cake -> loopy -> cake", idx.GetCircularReferences()[0].GenerateJourneyPath())
}

func TestExtractObject_DoubleRef_Circular_Direct_Fail(t *testing.T) {

	yml := `components:
  schemas:
    loopy:
      $ref: '#/components/schemas/cake'
    cake:
      $ref: '#/components/schemas/loopy'
    pizza:
      $ref: '#/components/schemas/cake'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// circular references are detected by the resolver, so lets run it!
	resolv := index.NewResolver(idx)
	assert.Len(t, resolv.CheckForCircularReferences(), 1)

	yml = `$ref: '#/components/schemas/why-did-westworld-have-to-end-so-poorly-ffs'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*pizza](context.Background(), "tags", cNode.Content[0], idx)
	assert.Error(t, err)

}

type test_borked struct {
	DontWork int
}

func (t test_borked) Build(_ context.Context, _, root *yaml.Node, idx *index.SpecIndex) error {
	return fmt.Errorf("I am always going to fail, every thing")
}

type test_noGood struct {
	DontWork int
}

func (t *test_noGood) Build(_ context.Context, _, root *yaml.Node, idx *index.SpecIndex) error {
	return fmt.Errorf("I am always going to fail a core build")
}

type test_almostGood struct {
	AlmostWork NodeReference[int]
}

func (t *test_almostGood) Build(_ context.Context, _, root *yaml.Node, idx *index.SpecIndex) error {
	return fmt.Errorf("I am always going to fail a build out")
}

type test_Good struct {
	AlmostWork NodeReference[int]
}

func (t *test_Good) Build(_ context.Context, _, root *yaml.Node, idx *index.SpecIndex) error {
	return nil
}

func TestExtractObject_BadLowLevelModel(t *testing.T) {

	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*test_noGood](context.Background(), "thing", &cNode, idx)
	assert.Error(t, err)

}

func TestExtractObject_BadBuild(t *testing.T) {

	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err := ExtractObject[*test_almostGood](context.Background(), "thing", &cNode, idx)
	assert.Error(t, err)

}

func TestExtractObject_BadLabel(t *testing.T) {

	yml := `components:
  schemas:
   hey:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `thing:
  dontWork: 123`
	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	res, err := ExtractObject[*test_almostGood](context.Background(), "ding", &cNode, idx)
	assert.Nil(t, res.Value)
	assert.NoError(t, err)

}

func TestExtractObject_PathIsCircular(t *testing.T) {

	// first we need an index.
	yml := `paths:
  '/something/here':
    post:
      $ref: '#/paths/~1something~1there/post'
  '/something/there':
    post:
      $ref: '#/paths/~1something~1here/post'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `thing:
  $ref: '#/paths/~1something~1here/post'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	res, err := ExtractObject[*test_Good](context.Background(), "thing", &rootNode, idx)
	assert.NotNil(t, res.Value)
	assert.Error(t, err) // circular error would have been thrown.

}

func TestExtractObject_PathIsCircular_IgnoreErrors(t *testing.T) {

	// first we need an index.
	yml := `paths:
  '/something/here':
    post:
      $ref: '#/paths/~1something~1there/post'
  '/something/there':
    post:
      $ref: '#/paths/~1something~1here/post'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	// disable circular ref checking.
	idx.SetAllowCircularReferenceResolving(true)

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `thing:
  $ref: '#/paths/~1something~1here/post'`

	var rootNode yaml.Node
	mErr = yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	res, err := ExtractObject[*test_Good](context.Background(), "thing", &rootNode, idx)
	assert.NotNil(t, res.Value)
	assert.NoError(t, err) // circular error would have been thrown, but we're ignoring them.

}

func TestExtractObjectRaw(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `description: hello pizza`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err, _, _ := ExtractObjectRaw[*pizza](context.Background(), nil, cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "hello pizza", tag.Description.Value)
}

func TestExtractObjectRaw_With_Ref(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err, isRef, rv := ExtractObjectRaw[*pizza](context.Background(), nil, cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, "hello", tag.Description.Value)
	assert.True(t, isRef)
	assert.Equal(t, "#/components/schemas/pizza", rv)
}

func TestExtractObjectRaw_Ref_Circular(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      $ref: '#/components/schemas/pie'
    pie:
      $ref: '#/components/schemas/pizza'`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err, _, _ := ExtractObjectRaw[*pizza](context.Background(), nil, cNode.Content[0], idx)
	assert.Error(t, err)
	assert.NotNil(t, tag)

}

func TestExtractObjectRaw_RefBroken(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hey!`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/lost-in-space'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	tag, err, _, _ := ExtractObjectRaw[*pizza](context.Background(), nil, cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Nil(t, tag)

}

func TestExtractObjectRaw_Ref_NonBuildable(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hey!`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `dontWork: 1'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err, _, _ := ExtractObjectRaw[*test_noGood](context.Background(), nil, cNode.Content[0], idx)
	assert.Error(t, err)

}

func TestExtractObjectRaw_Ref_AlmostBuildable(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hey!`
	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `almostWork: 1'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	_, err, _, _ := ExtractObjectRaw[*test_almostGood](context.Background(), nil, cNode.Content[0], idx)
	assert.Error(t, err)

}

func TestExtractArray(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: hello`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `things:
  - description: one
  - description: two
  - description: three`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*pizza](context.Background(), "things", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, things)
	assert.Equal(t, "one", things[0].Value.Description.Value)
	assert.Equal(t, "two", things[1].Value.Description.Value)
	assert.Equal(t, "three", things[2].Value.Description.Value)
}

func TestExtractArray_Ref(t *testing.T) {

	yml := `components:
  schemas:
    things:
      - description: one
      - description: two
      - description: three`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/things'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*pizza](context.Background(), "things", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, things)
	assert.Equal(t, "one", things[0].Value.Description.Value)
	assert.Equal(t, "two", things[1].Value.Description.Value)
	assert.Equal(t, "three", things[2].Value.Description.Value)
}

func TestExtractArray_Ref_Unbuildable(t *testing.T) {

	yml := `components:
  schemas:
    things:
      - description: one
      - description: two
      - description: three`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `$ref: '#/components/schemas/things'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_noGood](context.Background(), "", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_Ref_Circular(t *testing.T) {

	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/things'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_Good](context.Background(), "", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 2)
}

func TestExtractArray_Ref_Bad(t *testing.T) {

	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/let-us-eat-cake'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_Good](context.Background(), "", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_Ref_Nested(t *testing.T) {

	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `limes: 
  $ref: '#/components/schemas/let-us-eat-cake'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_Good](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_Ref_Nested_Circular(t *testing.T) {

	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `limes: 
  - $ref: '#/components/schemas/things'`

	var cNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &cNode)

	things, _, _, err := ExtractArray[*test_Good](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 1)
}

func TestExtractArray_Ref_Nested_BadRef(t *testing.T) {

	yml := `components:
  schemas:
    thongs:
      allOf:
        - $ref: '#/components/schemas/things'
    things:
      oneOf:
        - type: string`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `limes: 
  - $ref: '#/components/schemas/thangs'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_Good](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_Ref_Nested_CircularFlat(t *testing.T) {

	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `limes: 
  $ref: '#/components/schemas/things'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_Good](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 2)
}

func TestExtractArray_BadBuild(t *testing.T) {

	yml := `components:
  schemas:
    thongs:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `limes: 
  - dontWork: 1`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_noGood](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractArray_BadRefPropsTupe(t *testing.T) {

	yml := `components:
  parameters:
    cakes:
      limes: cake`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `limes: 
  $ref: '#/components/parameters/cakes'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)
	things, _, _, err := ExtractArray[*test_noGood](context.Background(), "limes", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractExample_String(t *testing.T) {
	yml := `hi`
	var e yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &e)

	exp := ExtractExample(e.Content[0], e.Content[0])
	assert.NotNil(t, exp.Value)
	assert.Equal(t, "hi", exp.Value)
}
func TestExtractExample_Map(t *testing.T) {
	yml := `one: two`
	var e yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &e)

	exp := ExtractExample(e.Content[0], e.Content[0])
	assert.NotNil(t, exp.Value)
	if n, ok := exp.Value.(map[string]interface{}); ok {
		assert.Equal(t, "two", n["one"])
	} else {
		panic("example unpacked incorrectly.")
	}
}

func TestExtractExample_Array(t *testing.T) {
	yml := `- hello`
	var e yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &e)

	exp := ExtractExample(e.Content[0], e.Content[0])
	assert.NotNil(t, exp.Value)
	if n, ok := exp.Value.([]interface{}); ok {
		assert.Equal(t, "hello", n[0])
	} else {
		panic("example unpacked incorrectly.")
	}
}

func TestExtractMapFlatNoLookup(t *testing.T) {

	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  description: two`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_Good](context.Background(), cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, things, 1)

}

func TestExtractMap_NoLookupWithExtensions(t *testing.T) {

	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  x-choo: choo`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookupExtensions[*test_Good](context.Background(), cNode.Content[0], idx, true)
	assert.NoError(t, err)
	assert.Len(t, things, 2)

	for k, v := range things {
		if k.Value == "x-hey" {
			continue
		}
		assert.Equal(t, "one", k.Value)
		assert.Len(t, v.ValueNode.Content, 2)
	}
}

func TestExtractMap_NoLookupWithExtensions_UsingMerge(t *testing.T) {

	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-yeah: &yeah
  night: fun
x-hey: you
one:
  x-choo: choo
<<: *yeah`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookupExtensions[*test_Good](context.Background(), cNode.Content[0], idx, true)
	assert.NoError(t, err)
	assert.Len(t, things, 4)

}

func TestExtractMap_NoLookupWithoutExtensions(t *testing.T) {

	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  x-choo: choo`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookupExtensions[*test_Good](context.Background(), cNode.Content[0], idx, false)
	assert.NoError(t, err)
	assert.Len(t, things, 1)

	for k := range things {
		assert.Equal(t, "one", k.Value)
	}
}

func TestExtractMap_WithExtensions(t *testing.T) {

	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  x-choo: choo`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMapExtensions[*test_Good](context.Background(), "one", cNode.Content[0], idx, true)
	assert.NoError(t, err)
	assert.Len(t, things, 1)
}

func TestExtractMap_WithoutExtensions(t *testing.T) {

	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  x-choo: choo`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMapExtensions[*test_Good](context.Background(), "one", cNode.Content[0], idx, false)
	assert.NoError(t, err)
	assert.Len(t, things, 0)
}

func TestExtractMapFlatNoLookup_Ref(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: tasty!`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_Good](context.Background(), cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, things, 1)

}

func TestExtractMapFlatNoLookup_Ref_Bad(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: tasty!`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/no-where-out-there'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_Good](context.Background(), cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)

}

func TestExtractMapFlatNoLookup_Ref_Circular(t *testing.T) {

	yml := `components:
  schemas:
    thongs:
      $ref: '#/components/schemas/things'
    things:
      $ref: '#/components/schemas/thongs'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/things'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_Good](context.Background(), cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 1)

}

func TestExtractMapFlatNoLookup_Ref_BadBuild(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      dontWork: 1`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
hello:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_noGood](context.Background(), cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)

}

func TestExtractMapFlatNoLookup_Ref_AlmostBuild(t *testing.T) {

	yml := `components:
  schemas:
    pizza:
      description: tasty!`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/pizza'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, err := ExtractMapNoLookup[*test_almostGood](context.Background(), cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)

}

func TestExtractMapFlat(t *testing.T) {

	yml := `components:`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  description: two`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, things, 1)

}

func TestExtractMapFlat_Ref(t *testing.T) {

	yml := `components:
  schemas:
    stank:
      things:
        almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `x-hey: you
one:
  $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, things, 1)

	for k := range things {
		assert.Equal(t, 99, things[k].Value.AlmostWork.Value)
	}

}

func TestExtractMapFlat_DoubleRef(t *testing.T) {

	yml := `components:
  schemas:
    stank:
      almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  nice:
    $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, things, 1)

	for k := range things {
		assert.Equal(t, 99, things[k].Value.AlmostWork.Value)
	}
}

func TestExtractMapFlat_DoubleRef_Error(t *testing.T) {

	yml := `components:
  schemas:
    stank:
      things:
        almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  nice:
    $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_almostGood](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)

}

func TestExtractMapFlat_DoubleRef_Error_NotFound(t *testing.T) {

	yml := `components:
  schemas:
    stank:
      things:
        almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  nice:
    $ref: '#/components/schemas/stanky-panky'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_almostGood](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)

}

func TestExtractMapFlat_DoubleRef_Circles(t *testing.T) {

	yml := `components:
  schemas:
    stonk:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/stonk'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `one:
  nice:
    $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 1)

}

func TestExtractMapFlat_Ref_Error(t *testing.T) {

	yml := `components:
  schemas:
    stank:
      x-smells: bad
      things:
        almostWork: 99`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_almostGood](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)

}

func TestExtractMapFlat_Ref_Circ_Error(t *testing.T) {

	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/stink'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 1)
}

func TestExtractMapFlat_Ref_Nested_Circ_Error(t *testing.T) {

	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/stink'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `one:
  $ref: '#/components/schemas/stank'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 1)
}

func TestExtractMapFlat_Ref_Nested_Error(t *testing.T) {

	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/none'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  $ref: '#/components/schemas/somewhere-else'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractMapFlat_BadKey_Ref_Nested_Error(t *testing.T) {

	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/none'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	yml = `one:
  $ref: '#/components/schemas/somewhere-else'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "not-even-there", cNode.Content[0], idx)
	assert.NoError(t, err)
	assert.Len(t, things, 0)
}

func TestExtractMapFlat_Ref_Bad(t *testing.T) {

	yml := `components:
  schemas:
    stink:
      $ref: '#/components/schemas/stank'
    stank:
      $ref: '#/components/schemas/none'`

	var idxNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &idxNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateClosedAPIIndexConfig())

	resolve := index.NewResolver(idx)
	errs := resolve.CheckForCircularReferences()
	assert.Len(t, errs, 1)

	yml = `$ref: '#/components/schemas/somewhere-else'`

	var cNode yaml.Node
	e := yaml.Unmarshal([]byte(yml), &cNode)
	assert.NoError(t, e)

	things, _, _, err := ExtractMap[*test_Good](context.Background(), "one", cNode.Content[0], idx)
	assert.Error(t, err)
	assert.Len(t, things, 0)
}

func TestExtractExtensions(t *testing.T) {

	yml := `x-bing: ding
x-bong: 1
x-ling: true
x-long: 0.99
x-fish:
  woo: yeah
x-tacos: [1,2,3]`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)

	r := ExtractExtensions(idxNode.Content[0])
	assert.Len(t, r, 6)
	for i := range r {
		switch i.Value {
		case "x-bing":
			assert.Equal(t, "ding", r[i].Value)
		case "x-bong":
			assert.Equal(t, int64(1), r[i].Value)
		case "x-ling":
			assert.Equal(t, true, r[i].Value)
		case "x-long":
			assert.Equal(t, 0.99, r[i].Value)
		case "x-fish":
			if a, ok := r[i].Value.(map[string]interface{}); ok {
				assert.Equal(t, "yeah", a["woo"])
			} else {
				panic("should not fail casting")
			}
		case "x-tacos":
			assert.Len(t, r[i].Value, 3)
		}
	}
}

type test_fresh struct {
	val   string
	thang *bool
}

func (f test_fresh) Hash() [32]byte {
	var data []string
	if f.val != "" {
		data = append(data, f.val)
	}
	if f.thang != nil {
		data = append(data, fmt.Sprintf("%v", *f.thang))
	}
	return sha256.Sum256([]byte(strings.Join(data, "|")))
}
func TestAreEqual(t *testing.T) {
	assert.True(t, AreEqual(test_fresh{val: "hello"}, test_fresh{val: "hello"}))
	assert.False(t, AreEqual(test_fresh{val: "hello"}, test_fresh{val: "goodbye"}))
	assert.False(t, AreEqual(nil, nil))
}

func TestGenerateHashString(t *testing.T) {

	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		GenerateHashString(test_fresh{val: "hello"}))

	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		GenerateHashString("hello"))

	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		GenerateHashString(""))

	assert.Equal(t, "",
		GenerateHashString(nil))

}

func TestGenerateHashString_Pointer(t *testing.T) {

	val := true
	assert.Equal(t, "b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b",
		GenerateHashString(test_fresh{thang: &val}))

	assert.Equal(t, "b5bea41b6c623f7c09f1bf24dcae58ebab3c0cdd90ad966bc43a45b44867e12b",
		GenerateHashString(&val))

}

func TestSetReference(t *testing.T) {

	type testObj struct {
		*Reference
	}

	n := testObj{Reference: &Reference{}}
	SetReference(&n, "#/pigeon/street")

	assert.Equal(t, "#/pigeon/street", n.GetReference())

}

func TestSetReference_nil(t *testing.T) {

	type testObj struct {
		*Reference
	}

	n := testObj{Reference: &Reference{}}
	SetReference(nil, "#/pigeon/street")
	assert.NotEqual(t, "#/pigeon/street", n.GetReference())
}

func TestLocateRefNode_CurrentPathKey_HttpLink(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "http://cakes.com#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "http://cakes.com#/components/schemas/thing")

	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_HttpLink_RemoteCtx(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "https://cakes.com#/components/schemas/thing")
	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_HttpLink_RemoteCtx_WithPath(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "https://cakes.com/jazzzy/shoes#/components/schemas/thing")
	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_Path_Link(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "yazzy.yaml#/components/schemas/thing",
			},
		},
	}

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "/jazzzy/shoes.yaml")
	idx := index.NewSpecIndexWithConfig(&no, index.CreateClosedAPIIndexConfig())
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_Path_URL(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "yazzy.yaml#/components/schemas/thing",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("https://herbs-and-coffee-in-the-fall.com")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	n, i, e, c := LocateRefNodeWithContext(context.Background(), &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_CurrentPathKey_DeeperPath_URL(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "slasshy/mazsshy/yazzy.yaml#/components/schemas/thing",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("https://herbs-and-coffee-in-the-fall.com/pizza/burgers")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	n, i, e, c := LocateRefNodeWithContext(context.Background(), &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_NoExplode(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "components/schemas/thing.yaml",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smiledfdfdfdfds.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	n, i, e, c := LocateRefNodeWithContext(context.Background(), &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_NoExplode_HTTP(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "components/schemas/thing.yaml",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smilfghfhfhfhfhes.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "http://minty-fresh-shoes.com/nice/no.yaml")
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_NoExplode_NoSpecPath(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "components/schemas/thing.yaml",
			},
		},
	}

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smilfghfhfhfhfhes.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&no, cf)
	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "no.yaml")
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.Nil(t, n)
	assert.NotNil(t, i)
	assert.NotNil(t, e)
	assert.NotNil(t, c)
}

func TestLocateRefNode_DoARealLookup(t *testing.T) {

	no := yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "$ref",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "/root.yaml#/components/schemas/Burger",
			},
		},
	}

	b, err := os.ReadFile("../../test_specs/burgershop.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var rootNode yaml.Node
	_ = yaml.Unmarshal(b, &rootNode)

	cf := index.CreateClosedAPIIndexConfig()
	u, _ := url.Parse("http://smilfghfhfhfhfhes.com/bikes")
	cf.BaseURL = u
	idx := index.NewSpecIndexWithConfig(&rootNode, cf)

	// fake cache to a lookup for a file that does not exist will work.
	fakeCache := new(syncmap.Map)
	fakeCache.Store("/root.yaml#/components/schemas/Burger", &index.Reference{Node: &no, Index: idx})
	idx.SetCache(fakeCache)

	ctx := context.WithValue(context.Background(), index.CurrentPathKey, "/root.yaml#/components/schemas/Burger")
	n, i, e, c := LocateRefNodeWithContext(ctx, &no, idx)
	assert.NotNil(t, n)
	assert.NotNil(t, i)
	assert.Nil(t, e)
	assert.NotNil(t, c)
}
