// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package gateway_target

import (
	"encoding/json"
	"fmt"

	svcapitypes "github.com/aws-controllers-k8s/bedrockagentcorecontrol-controller/apis/v1alpha1"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol/types"
)

// stringToSchemaDefinition unmarshals a JSON string into an SDK
// SchemaDefinition. The SchemaDefinition type is recursive (it contains
// Items *SchemaDefinition and Properties map[string]SchemaDefinition) which
// cannot be directly represented in a CRD. We accept it as a JSON string
// from the user and unmarshal it here.
func stringToSchemaDefinition(s *string) (*svcsdktypes.SchemaDefinition, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	var sd svcsdktypes.SchemaDefinition
	if err := json.Unmarshal([]byte(*s), &sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SchemaDefinition JSON: %w", err)
	}
	return &sd, nil
}

// getMcpLambdaInlinePayload returns the inline payload ToolDefinitions from
// the TargetConfiguration if the target is an MCP Lambda target with inline
// payload tool schema. Returns nil if the path doesn't apply.
func getMcpLambdaInlinePayload(tc *svcapitypes.TargetConfiguration) []*svcapitypes.ToolDefinition {
	if tc == nil || tc.Mcp == nil || tc.Mcp.Lambda == nil ||
		tc.Mcp.Lambda.ToolSchema == nil || tc.Mcp.Lambda.ToolSchema.InlinePayload == nil {
		return nil
	}
	return tc.Mcp.Lambda.ToolSchema.InlinePayload
}

// getSDKInlinePayload extracts the InlinePayload ToolDefinition slice from an
// SDK TargetConfiguration, returning nil if the path doesn't match.
func getSDKInlinePayload(tc svcsdktypes.TargetConfiguration) []svcsdktypes.ToolDefinition {
	mcpMember, ok := tc.(*svcsdktypes.TargetConfigurationMemberMcp)
	if !ok || mcpMember == nil {
		return nil
	}
	lambdaMember, ok := mcpMember.Value.(*svcsdktypes.McpTargetConfigurationMemberLambda)
	if !ok || lambdaMember == nil {
		return nil
	}
	inlinePayloadMember, ok := lambdaMember.Value.ToolSchema.(*svcsdktypes.ToolSchemaMemberInlinePayload)
	if !ok || inlinePayloadMember == nil {
		return nil
	}
	return inlinePayloadMember.Value
}

// setSchemaDefinitionsOnInput walks the SDK input's ToolDefinition slice and
// sets InputSchema/OutputSchema by unmarshaling the JSON strings from the CR.
func setSchemaDefinitionsOnInput(
	toolDefs []*svcapitypes.ToolDefinition,
	sdkToolDefs []svcsdktypes.ToolDefinition,
) error {
	for i, td := range toolDefs {
		if i >= len(sdkToolDefs) {
			break
		}
		if td.InputSchema != nil {
			sd, err := stringToSchemaDefinition(td.InputSchema)
			if err != nil {
				return err
			}
			sdkToolDefs[i].InputSchema = sd
		}
		if td.OutputSchema != nil {
			sd, err := stringToSchemaDefinition(td.OutputSchema)
			if err != nil {
				return err
			}
			sdkToolDefs[i].OutputSchema = sd
		}
	}
	return nil
}

// setSchemaDefinitionsOnCreateInput unmarshals the InputSchema and OutputSchema
// JSON strings from the custom resource into SDK SchemaDefinition objects on
// the CreateGatewayTarget input.
func setSchemaDefinitionsOnCreateInput(desired *resource, input *svcsdk.CreateGatewayTargetInput) error {
	toolDefs := getMcpLambdaInlinePayload(desired.ko.Spec.TargetConfiguration)
	if toolDefs == nil {
		return nil
	}
	sdkToolDefs := getSDKInlinePayload(input.TargetConfiguration)
	if sdkToolDefs == nil {
		return nil
	}
	return setSchemaDefinitionsOnInput(toolDefs, sdkToolDefs)
}

// setSchemaDefinitionsOnUpdateInput unmarshals the InputSchema and OutputSchema
// JSON strings from the custom resource into SDK SchemaDefinition objects on
// the UpdateGatewayTarget input.
func setSchemaDefinitionsOnUpdateInput(desired *resource, input *svcsdk.UpdateGatewayTargetInput) error {
	toolDefs := getMcpLambdaInlinePayload(desired.ko.Spec.TargetConfiguration)
	if toolDefs == nil {
		return nil
	}
	sdkToolDefs := getSDKInlinePayload(input.TargetConfiguration)
	if sdkToolDefs == nil {
		return nil
	}
	return setSchemaDefinitionsOnInput(toolDefs, sdkToolDefs)
}
