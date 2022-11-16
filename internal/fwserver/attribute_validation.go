package fwserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/internal/fwschema"
	"github.com/hashicorp/terraform-plugin-framework/internal/fwschema/fwxschema"
	"github.com/hashicorp/terraform-plugin-framework/internal/fwschemadata"
	"github.com/hashicorp/terraform-plugin-framework/internal/logging"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AttributeValidate performs all Attribute validation.
//
// TODO: Clean up this abstraction back into an internal Attribute type method.
// The extra Attribute parameter is a carry-over of creating the proto6server
// package from the tfsdk package and not wanting to export the method.
// Reference: https://github.com/hashicorp/terraform-plugin-framework/issues/365
func AttributeValidate(ctx context.Context, a fwschema.Attribute, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	ctx = logging.FrameworkWithAttributePath(ctx, req.AttributePath.String())

	tfsdkAttribute, ok := a.(tfsdk.Attribute) //nolint:staticcheck // Handle tfsdk.Attribute until its removed.

	if ok && tfsdkAttribute.GetType() == nil {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid Attribute Definition",
			"Attribute must define either Attributes or Type. This is always a problem with the provider and should be reported to the provider developer.",
		)

		return
	}

	if ok && len(tfsdkAttribute.GetAttributes()) > 0 && tfsdkAttribute.GetNestingMode() == fwschema.NestingModeUnknown {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid Attribute Definition",
			"Attribute cannot define both Attributes and Type. This is always a problem with the provider and should be reported to the provider developer.",
		)

		return
	}

	if !a.IsRequired() && !a.IsOptional() && !a.IsComputed() {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid Attribute Definition",
			"Attribute missing Required, Optional, or Computed definition. This is always a problem with the provider and should be reported to the provider developer.",
		)

		return
	}

	configData := &fwschemadata.Data{
		Description:    fwschemadata.DataDescriptionConfiguration,
		Schema:         req.Config.Schema,
		TerraformValue: req.Config.Raw,
	}

	attributeConfig, diags := configData.ValueAtPath(ctx, req.AttributePath)
	resp.Diagnostics.Append(diags...)

	if diags.HasError() {
		return
	}

	// Terraform CLI does not automatically perform certain configuration
	// checks yet. If it eventually does, this logic should remain at least
	// until Terraform CLI versions 0.12 through the release containing the
	// checks are considered end-of-life.
	// Reference: https://github.com/hashicorp/terraform/issues/30669
	if a.IsComputed() && !a.IsOptional() && !attributeConfig.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Invalid Configuration for Read-Only Attribute",
			"Cannot set value for this attribute as the provider has marked it as read-only. Remove the configuration line setting the value.\n\n"+
				"Refer to the provider documentation or contact the provider developers for additional information about configurable and read-only attributes that are supported.",
		)
	}

	// Terraform CLI does not automatically perform certain configuration
	// checks yet. If it eventually does, this logic should remain at least
	// until Terraform CLI versions 0.12 through the release containing the
	// checks are considered end-of-life.
	// Reference: https://github.com/hashicorp/terraform/issues/30669
	if a.IsRequired() && attributeConfig.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Missing Configuration for Required Attribute",
			fmt.Sprintf("Must set a configuration value for the %s attribute as the provider has marked it as required.\n\n", req.AttributePath.String())+
				"Refer to the provider documentation or contact the provider developers for additional information about configurable attributes that are required.",
		)
	}

	req.AttributeConfig = attributeConfig

	if attributeWithValidators, ok := a.(fwxschema.AttributeWithValidators); ok {
		for _, validator := range attributeWithValidators.GetValidators() {
			logging.FrameworkDebug(
				ctx,
				"Calling provider defined AttributeValidator",
				map[string]interface{}{
					logging.KeyDescription: validator.Description(ctx),
				},
			)
			validator.Validate(ctx, req, resp)
			logging.FrameworkDebug(
				ctx,
				"Called provider defined AttributeValidator",
				map[string]interface{}{
					logging.KeyDescription: validator.Description(ctx),
				},
			)
		}
	}

	AttributeValidateNestedAttributes(ctx, a, req, resp)

	// Show deprecation warnings only for known values.
	if a.GetDeprecationMessage() != "" && !attributeConfig.IsNull() && !attributeConfig.IsUnknown() {
		resp.Diagnostics.AddAttributeWarning(
			req.AttributePath,
			"Attribute Deprecated",
			a.GetDeprecationMessage(),
		)
	}
}

// AttributeValidateNestedAttributes performs all nested Attributes validation.
//
// TODO: Clean up this abstraction back into an internal Attribute type method.
// The extra Attribute parameter is a carry-over of creating the proto6server
// package from the tfsdk package and not wanting to export the method.
// Reference: https://github.com/hashicorp/terraform-plugin-framework/issues/365
func AttributeValidateNestedAttributes(ctx context.Context, a fwschema.Attribute, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	nestedAttribute, ok := a.(fwschema.NestedAttribute)

	if !ok {
		return
	}

	tfsdkAttribute, ok := a.(tfsdk.Attribute) //nolint:staticcheck // Handle tfsdk.Attribute until its removed.

	if ok && tfsdkAttribute.GetNestingMode() == fwschema.NestingModeUnknown {
		return
	}

	nm := nestedAttribute.GetNestingMode()
	switch nm {
	case fwschema.NestingModeList:
		listVal, ok := req.AttributeConfig.(types.ListValuable)

		if !ok {
			err := fmt.Errorf("unknown attribute value type (%T) for nesting mode (%T) at path: %s", req.AttributeConfig, nm, req.AttributePath)
			resp.Diagnostics.AddAttributeError(
				req.AttributePath,
				"Attribute Validation Error Invalid Value Type",
				"A type that implements types.ListValuable is expected here. Report this to the provider developer:\n\n"+err.Error(),
			)

			return
		}

		l, diags := listVal.ToListValue(ctx)

		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for idx := range l.Elements() {
			for nestedName, nestedAttr := range nestedAttribute.GetAttributes() {
				nestedAttrReq := tfsdk.ValidateAttributeRequest{
					AttributePath:           req.AttributePath.AtListIndex(idx).AtName(nestedName),
					AttributePathExpression: req.AttributePathExpression.AtListIndex(idx).AtName(nestedName),
					Config:                  req.Config,
				}
				nestedAttrResp := &tfsdk.ValidateAttributeResponse{
					Diagnostics: resp.Diagnostics,
				}

				AttributeValidate(ctx, nestedAttr, nestedAttrReq, nestedAttrResp)

				resp.Diagnostics = nestedAttrResp.Diagnostics
			}
		}
	case fwschema.NestingModeSet:
		setVal, ok := req.AttributeConfig.(types.SetValuable)

		if !ok {
			err := fmt.Errorf("unknown attribute value type (%T) for nesting mode (%T) at path: %s", req.AttributeConfig, nm, req.AttributePath)
			resp.Diagnostics.AddAttributeError(
				req.AttributePath,
				"Attribute Validation Error Invalid Value Type",
				"A type that implements types.SetValuable is expected here. Report this to the provider developer:\n\n"+err.Error(),
			)

			return
		}

		s, diags := setVal.ToSetValue(ctx)

		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, value := range s.Elements() {
			for nestedName, nestedAttr := range nestedAttribute.GetAttributes() {
				nestedAttrReq := tfsdk.ValidateAttributeRequest{
					AttributePath:           req.AttributePath.AtSetValue(value).AtName(nestedName),
					AttributePathExpression: req.AttributePathExpression.AtSetValue(value).AtName(nestedName),
					Config:                  req.Config,
				}
				nestedAttrResp := &tfsdk.ValidateAttributeResponse{
					Diagnostics: resp.Diagnostics,
				}

				AttributeValidate(ctx, nestedAttr, nestedAttrReq, nestedAttrResp)

				resp.Diagnostics = nestedAttrResp.Diagnostics
			}
		}
	case fwschema.NestingModeMap:
		mapVal, ok := req.AttributeConfig.(types.MapValuable)

		if !ok {
			err := fmt.Errorf("unknown attribute value type (%T) for nesting mode (%T) at path: %s", req.AttributeConfig, nm, req.AttributePath)
			resp.Diagnostics.AddAttributeError(
				req.AttributePath,
				"Attribute Validation Error Invalid Value Type",
				"A type that implements types.MapValuable is expected here. Report this to the provider developer:\n\n"+err.Error(),
			)

			return
		}

		m, diags := mapVal.ToMapValue(ctx)

		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for key := range m.Elements() {
			for nestedName, nestedAttr := range nestedAttribute.GetAttributes() {
				nestedAttrReq := tfsdk.ValidateAttributeRequest{
					AttributePath:           req.AttributePath.AtMapKey(key).AtName(nestedName),
					AttributePathExpression: req.AttributePathExpression.AtMapKey(key).AtName(nestedName),
					Config:                  req.Config,
				}
				nestedAttrResp := &tfsdk.ValidateAttributeResponse{
					Diagnostics: resp.Diagnostics,
				}

				AttributeValidate(ctx, nestedAttr, nestedAttrReq, nestedAttrResp)

				resp.Diagnostics = nestedAttrResp.Diagnostics
			}
		}
	case fwschema.NestingModeSingle:
		objectVal, ok := req.AttributeConfig.(types.ObjectValuable)

		if !ok {
			err := fmt.Errorf("unknown attribute value type (%T) for nesting mode (%T) at path: %s", req.AttributeConfig, nm, req.AttributePath)
			resp.Diagnostics.AddAttributeError(
				req.AttributePath,
				"Attribute Validation Error Invalid Value Type",
				"A type that implements types.ObjectValuable is expected here. Report this to the provider developer:\n\n"+err.Error(),
			)

			return
		}

		o, diags := objectVal.ToObjectValue(ctx)

		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if o.IsNull() || o.IsUnknown() {
			return
		}

		for nestedName, nestedAttr := range nestedAttribute.GetAttributes() {
			nestedAttrReq := tfsdk.ValidateAttributeRequest{
				AttributePath:           req.AttributePath.AtName(nestedName),
				AttributePathExpression: req.AttributePathExpression.AtName(nestedName),
				Config:                  req.Config,
			}
			nestedAttrResp := &tfsdk.ValidateAttributeResponse{
				Diagnostics: resp.Diagnostics,
			}

			AttributeValidate(ctx, nestedAttr, nestedAttrReq, nestedAttrResp)

			resp.Diagnostics = nestedAttrResp.Diagnostics
		}
	default:
		err := fmt.Errorf("unknown attribute validation nesting mode (%T: %v) at path: %s", nm, nm, req.AttributePath)
		resp.Diagnostics.AddAttributeError(
			req.AttributePath,
			"Attribute Validation Error",
			"Attribute validation cannot walk schema. Report this to the provider developer:\n\n"+err.Error(),
		)

		return
	}
}
