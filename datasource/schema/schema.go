package schema

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/internal/fwschema"
	"github.com/hashicorp/terraform-plugin-framework/internal/totftypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Schema must satify the fwschema.Schema interface.
var _ fwschema.Schema = Schema{}

type Schema struct {
	Attributes          map[string]Attribute
	Blocks              map[string]Block
	Description         string
	MarkdownDescription string
	DeprecationMessage  string
}

// ApplyTerraform5AttributePathStep applies the given AttributePathStep to the
// schema.
func (s Schema) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (interface{}, error) {
	a, ok := step.(tftypes.AttributeName)

	if !ok {
		return nil, fmt.Errorf("cannot apply AttributePathStep %T to schema", step)
	}

	attrName := string(a)

	if attr, ok := s.Attributes[attrName]; ok {
		return attr, nil
	}

	if block, ok := s.Blocks[attrName]; ok {
		return block, nil
	}

	return nil, fmt.Errorf("could not find attribute or block %q in schema", a)
}

// AttributeAtPath returns the Attribute at the passed path. If the path points
// to an element or attribute of a complex type, rather than to an Attribute,
// it will return an ErrPathInsideAtomicAttribute error.
func (s Schema) AttributeAtPath(ctx context.Context, schemaPath path.Path) (fwschema.Attribute, diag.Diagnostics) {
	var diags diag.Diagnostics

	tftypesPath, tftypesDiags := totftypes.AttributePath(ctx, schemaPath)

	diags.Append(tftypesDiags...)

	if diags.HasError() {
		return nil, diags
	}

	attribute, err := s.AttributeAtTerraformPath(ctx, tftypesPath)

	if err != nil {
		diags.AddAttributeError(
			schemaPath,
			"Invalid Schema Path",
			"When attempting to get the framework attribute associated with a schema path, an unexpected error was returned. "+
				"This is always an issue with the provider. Please report this to the provider developers.\n\n"+
				fmt.Sprintf("Path: %s\n", schemaPath.String())+
				fmt.Sprintf("Original Error: %s", err),
		)
		return nil, diags
	}

	return attribute, diags
}

// AttributeAtPath returns the Attribute at the passed path. If the path points
// to an element or attribute of a complex type, rather than to an Attribute,
// it will return an ErrPathInsideAtomicAttribute error.
func (s Schema) AttributeAtTerraformPath(_ context.Context, path *tftypes.AttributePath) (fwschema.Attribute, error) {
	res, remaining, err := tftypes.WalkAttributePath(s, path)

	if err != nil {
		return nil, fmt.Errorf("%v still remains in the path: %w", remaining, err)
	}

	switch r := res.(type) {
	case attr.Type:
		return nil, fwschema.ErrPathInsideAtomicAttribute
	case fwschema.UnderlyingAttributes:
		return nil, fwschema.ErrPathInsideAtomicAttribute
	case fwschema.NestedBlock:
		return nil, fwschema.ErrPathInsideAtomicAttribute
	case Attribute:
		return r, nil
	case Block:
		return nil, fwschema.ErrPathIsBlock
	default:
		return nil, fmt.Errorf("Schema AttributeAtTerraformPath got unexpected type %T", res)
	}
}

// GetAttributes returns the Attributes field value.
func (s Schema) GetAttributes() map[string]fwschema.Attribute {
	return schemaAttributes(s.Attributes)
}

// GetBlocks returns the Blocks field value.
func (s Schema) GetBlocks() map[string]fwschema.Block {
	return schemaBlocks(s.Blocks)
}

// GetDeprecationMessage returns the DeprecationMessage field value.
func (s Schema) GetDeprecationMessage() string {
	return s.DeprecationMessage
}

// GetDescription returns the Description field value.
func (s Schema) GetDescription() string {
	return s.Description
}

// GetMarkdownDescription returns the MarkdownDescription field value.
func (s Schema) GetMarkdownDescription() string {
	return s.MarkdownDescription
}

// GetVersion always returns 0 as datasource schemas cannot be versioned.
func (s Schema) GetVersion() int64 {
	return 0
}

// Type returns the framework type of the schema.
func (s Schema) Type() attr.Type {
	attrTypes := map[string]attr.Type{}

	for name, attr := range s.Attributes {
		attrTypes[name] = attr.GetType()
	}

	for name, block := range s.Blocks {
		attrTypes[name] = block.Type()
	}

	return types.ObjectType{AttrTypes: attrTypes}
}

// TypeAtPath returns the framework type at the given schema path.
func (s Schema) TypeAtPath(ctx context.Context, schemaPath path.Path) (attr.Type, diag.Diagnostics) {
	var diags diag.Diagnostics

	tftypesPath, tftypesDiags := totftypes.AttributePath(ctx, schemaPath)

	diags.Append(tftypesDiags...)

	if diags.HasError() {
		return nil, diags
	}

	attrType, err := s.TypeAtTerraformPath(ctx, tftypesPath)

	if err != nil {
		diags.AddAttributeError(
			schemaPath,
			"Invalid Schema Path",
			"When attempting to get the framework type associated with a schema path, an unexpected error was returned. "+
				"This is always an issue with the provider. Please report this to the provider developers.\n\n"+
				fmt.Sprintf("Path: %s\n", schemaPath.String())+
				fmt.Sprintf("Original Error: %s", err),
		)
		return nil, diags
	}

	return attrType, diags
}

// TypeAtTerraformPath returns the framework type at the given tftypes path.
func (s Schema) TypeAtTerraformPath(_ context.Context, path *tftypes.AttributePath) (attr.Type, error) {
	rawType, remaining, err := tftypes.WalkAttributePath(s, path)
	if err != nil {
		return nil, fmt.Errorf("%v still remains in the path: %w", remaining, err)
	}

	switch typ := rawType.(type) {
	case attr.Type:
		return typ, nil
	case fwschema.UnderlyingAttributes:
		return typ.Type(), nil
	case fwschema.Attribute:
		return typ.GetType(), nil
	case fwschema.NestedBlock:
		return typ.Block.Type(), nil
	case fwschema.Block:
		return typ.Type(), nil
	case Schema:
		return typ.Type(), nil
	default:
		return nil, fmt.Errorf("Schema TypeAtTerraformPath got unexpected type %T", rawType)
	}
}

// schemaAttributes is a datasource to fwschema type conversion function.
func schemaAttributes(attributes map[string]Attribute) map[string]fwschema.Attribute {
	result := make(map[string]fwschema.Attribute, len(attributes))

	for name, attribute := range attributes {
		result[name] = attribute
	}

	return result
}

// schemaBlocks is a datasource to fwschema type conversion function.
func schemaBlocks(blocks map[string]Block) map[string]fwschema.Block {
	result := make(map[string]fwschema.Block, len(blocks))

	for name, block := range blocks {
		result[name] = block
	}

	return result
}
