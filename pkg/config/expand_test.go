package config

import (
	"fmt"
	"hpc-toolkit/pkg/resreader"

	. "gopkg.in/check.v1"
)

func (s *MySuite) TestGetResourceVarName(c *C) {
	resID := "resID"
	varName := "varName"
	expected := fmt.Sprintf("$(%s.%s)", resID, varName)
	got := getResourceVarName(resID, varName)
	c.Assert(got, Equals, expected)
}

func (s *MySuite) TestUpdateVariableType(c *C) {
	// slice, success
	// empty
	testSlice := []interface{}{}
	ctx := varContext{}
	resToGrp := make(map[string]int)
	ret, err := updateVariableType(testSlice, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testSlice, DeepEquals, ret)
	// single string
	testSlice = append(testSlice, "string")
	ret, err = updateVariableType(testSlice, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testSlice, DeepEquals, ret)
	// add list
	testSlice = append(testSlice, []interface{}{})
	ret, err = updateVariableType(testSlice, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testSlice, DeepEquals, ret)
	// add map
	testSlice = append(testSlice, make(map[string]interface{}))
	ret, err = updateVariableType(testSlice, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testSlice, DeepEquals, ret)

	// map, success
	testMap := make(map[string]interface{})
	ret, err = updateVariableType(testMap, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testMap, DeepEquals, ret)
	// add string
	testMap["string"] = "string"
	ret, err = updateVariableType(testMap, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testMap, DeepEquals, ret)
	// add map
	testMap["map"] = make(map[string]interface{})
	ret, err = updateVariableType(testMap, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testMap, DeepEquals, ret)
	// add slice
	testMap["slice"] = []interface{}{}
	ret, err = updateVariableType(testMap, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testMap, DeepEquals, ret)

	// string, success
	testString := "string"
	ret, err = updateVariableType(testString, ctx, resToGrp)
	c.Assert(err, IsNil)
	c.Assert(testString, DeepEquals, ret)
}

func (s *MySuite) TestCombineLabels(c *C) {
	bc := getBlueprintConfigForTest()

	err := bc.combineLabels()
	c.Assert(err, IsNil)

	// Were global labels created?
	_, exists := bc.Config.Vars["labels"]
	c.Assert(exists, Equals, true)

	// Was the ghpc_blueprint label set correctly?
	globalLabels := bc.Config.Vars["labels"].(map[string]interface{})
	ghpcBlueprint, exists := globalLabels[blueprintLabel]
	c.Assert(exists, Equals, true)
	c.Assert(ghpcBlueprint, Equals, bc.Config.BlueprintName)

	// Was the ghpc_deployment label set correctly?
	ghpcDeployment, exists := globalLabels[deploymentLabel]
	c.Assert(exists, Equals, true)
	c.Assert(ghpcDeployment, Equals, "undefined")

	// Was "labels" created for the resource with no settings?
	_, exists = bc.Config.ResourceGroups[0].Resources[0].Settings["labels"]
	c.Assert(exists, Equals, true)

	resourceLabels := bc.Config.ResourceGroups[0].Resources[0].
		Settings["labels"].(map[interface{}]interface{})

	// Was the role created correctly?
	ghpcRole, exists := resourceLabels[roleLabel]
	c.Assert(exists, Equals, true)
	c.Assert(ghpcRole, Equals, "other")

	// Test invalid labels
	bc.Config.Vars["labels"] = "notAMap"
	err = bc.combineLabels()
	expectedErrorStr := fmt.Sprintf("%s: found %T",
		errorMessages["globalLabelType"], bc.Config.Vars["labels"])
	c.Assert(err, ErrorMatches, expectedErrorStr)

}

func (s *MySuite) TestApplyGlobalVariables(c *C) {
	bc := getBlueprintConfigForTest()
	testResource := bc.Config.ResourceGroups[0].Resources[0]

	// Test no inputs, none required
	err := bc.applyGlobalVariables()
	c.Assert(err, IsNil)

	// Test no inputs, one required, doesn't exist in globals
	bc.ResourcesInfo["group1"][testResource.Source] = resreader.ResourceInfo{
		Inputs: []resreader.VarInfo{requiredVar},
	}
	err = bc.applyGlobalVariables()
	expectedErrorStr := fmt.Sprintf("%s: Resource.ID: %s Setting: %s",
		errorMessages["missingSetting"], testResource.ID, requiredVar.Name)
	c.Assert(err, ErrorMatches, expectedErrorStr)

	// Test no input, one required, exists in globals
	bc.Config.Vars[requiredVar.Name] = "val"
	err = bc.applyGlobalVariables()
	c.Assert(err, IsNil)
	c.Assert(
		bc.Config.ResourceGroups[0].Resources[0].Settings[requiredVar.Name],
		Equals, fmt.Sprintf("((var.%s))", requiredVar.Name))

	// Test one input, one required
	bc.Config.ResourceGroups[0].Resources[0].Settings[requiredVar.Name] = "val"
	err = bc.applyGlobalVariables()
	c.Assert(err, IsNil)

	// Test one input, none required, exists in globals
	bc.ResourcesInfo["group1"][testResource.Source].Inputs[0].Required = false
	err = bc.applyGlobalVariables()
	c.Assert(err, IsNil)
}

func (s *MySuite) TestIsSimpleVariable(c *C) {
	// True: Correct simple variable
	got := isSimpleVariable("$(some_text)")
	c.Assert(got, Equals, true)
	// False: Missing $
	got = isSimpleVariable("(some_text)")
	c.Assert(got, Equals, false)
	// False: Missing (
	got = isSimpleVariable("$some_text)")
	c.Assert(got, Equals, false)
	// False: Missing )
	got = isSimpleVariable("$(some_text")
	c.Assert(got, Equals, false)
	// False: Contains Prefix
	got = isSimpleVariable("prefix-$(some_text)")
	c.Assert(got, Equals, false)
	// False: Contains Suffix
	got = isSimpleVariable("$(some_text)-suffix")
	c.Assert(got, Equals, false)
	// False: Contains prefix and suffix
	got = isSimpleVariable("prefix-$(some_text)-suffix")
	c.Assert(got, Equals, false)
	// False: empty string
	got = isSimpleVariable("")
	c.Assert(got, Equals, false)
}

func (s *MySuite) TestHasVariable(c *C) {
	// True: simple variable
	got := hasVariable("$(some_text)")
	c.Assert(got, Equals, true)
	// True: has prefix
	got = hasVariable("prefix-$(some_text)")
	c.Assert(got, Equals, true)
	// True: has suffix
	got = hasVariable("$(some_text)-suffix")
	c.Assert(got, Equals, true)
	// True: Two variables
	got = hasVariable("$(some_text)$(some_more)")
	c.Assert(got, Equals, true)
	// True: two variable with other text
	got = hasVariable("prefix-$(some_text)-$(some_more)-suffix")
	c.Assert(got, Equals, true)
	// False: missing $
	got = hasVariable("(some_text)")
	c.Assert(got, Equals, false)
	// False: missing (
	got = hasVariable("$some_text)")
	c.Assert(got, Equals, false)
	// False: missing )
	got = hasVariable("$(some_text")
	c.Assert(got, Equals, false)
}

func (s *MySuite) TestExpandSimpleVariable(c *C) {
	// Setup
	testResID := "existingResource"
	testResource := Resource{
		ID:     testResID,
		Kind:   "terraform",
		Source: "./resource/testpath",
	}
	testYamlConfig := YamlConfig{
		Vars: make(map[string]interface{}),
		ResourceGroups: []ResourceGroup{
			ResourceGroup{
				Resources: []Resource{
					testResource,
				},
			},
		},
	}
	testVarContext := varContext{
		yamlConfig: testYamlConfig,
		resIndex:   0,
		groupIndex: 0,
	}
	testResToGrp := make(map[string]int)

	// Invalid variable -> no .
	testVarContext.varString = "$(varsStringWithNoDot)"
	_, err := expandSimpleVariable(testVarContext, testResToGrp)
	expectedErr := fmt.Sprintf("%s.*", errorMessages["invalidVar"])
	c.Assert(err, ErrorMatches, expectedErr)

	// Global variable: Invalid -> not found
	testVarContext.varString = "$(vars.doesntExists)"
	_, err = expandSimpleVariable(testVarContext, testResToGrp)
	expectedErr = fmt.Sprintf("%s: .*", errorMessages["varNotFound"])
	c.Assert(err, ErrorMatches, expectedErr)

	// Global variable: Success
	testVarContext.yamlConfig.Vars["globalExists"] = "existsValue"
	testVarContext.varString = "$(vars.globalExists)"
	got, err := expandSimpleVariable(testVarContext, testResToGrp)
	c.Assert(err, IsNil)
	expected := "((var.globalExists))"
	c.Assert(got, Equals, expected)

	// Resource variable: Invalid -> Resource not found
	testVarContext.varString = "$(notARes.someVar)"
	_, err = expandSimpleVariable(testVarContext, testResToGrp)
	expectedErr = fmt.Sprintf("%s: .*", errorMessages["varNotFound"])
	c.Assert(err, ErrorMatches, expectedErr)

	// Resource variable: Invalid -> Output not found
	reader := resreader.Factory("terraform")
	reader.SetInfo(testResource.Source, resreader.ResourceInfo{})
	testResToGrp[testResID] = 0
	fakeOutput := "doesntExist"
	testVarContext.varString = fmt.Sprintf("$(%s.%s)", testResource.ID, fakeOutput)
	_, err = expandSimpleVariable(testVarContext, testResToGrp)
	expectedErr = fmt.Sprintf("%s: resource %s did not have output %s",
		errorMessages["noOutput"], testResID, fakeOutput)
	c.Assert(err, ErrorMatches, expectedErr)

	// Resource variable: Success
	existingOutput := "outputExists"
	testVarInfoOutput := resreader.VarInfo{Name: existingOutput}
	testResInfo := resreader.ResourceInfo{
		Outputs: []resreader.VarInfo{testVarInfoOutput},
	}
	reader.SetInfo(testResource.Source, testResInfo)
	testVarContext.varString = fmt.Sprintf(
		"$(%s.%s)", testResource.ID, existingOutput)
	got, err = expandSimpleVariable(testVarContext, testResToGrp)
	c.Assert(err, IsNil)
	expected = fmt.Sprintf("((module.%s.%s))", testResource.ID, existingOutput)
	c.Assert(got, Equals, expected)
}
