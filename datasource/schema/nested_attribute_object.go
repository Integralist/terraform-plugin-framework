package schema

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// NestedAttributeObject is the object containing the underlying attributes
// for a ListNestedAttribute, MapNestedAttribute, or SetNestedAttribute. When
// retrieving the value for this attribute, use types.Object as the value type
// unless the CustomType field is set. The Attributes field must be set. Nested
// attributes are only compatible with protocol version 6.
//
// This object enables customizing and simplifying details within its parent
// NestedAttribute, therefore it cannot have Terraform schema fields such as
// Required, Description, etc.
type NestedAttributeObject struct {
	// Attributes is the mapping of underlying attribute names to attribute
	// definitions. This field must be set.
	Attributes map[string]Attribute

	// CustomType enables the use of a custom attribute type in place of the
	// default types.ObjectType. When retrieving data, the types.ObjectValuable
	// associated with this custom type must be used in place of types.Object.
	CustomType types.ObjectTypable

	//Validators          []StringValidator
}

// ApplyTerraform5AttributePathStep performs an AttributeName step on the
// underlying attributes or returns an error.
func (o NestedAttributeObject) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (any, error) {
	name, ok := step.(tftypes.AttributeName)

	if !ok {
		return nil, fmt.Errorf("can't apply %T to NestedAttributeObject", step)
	}

	attribute, ok := o.Attributes[string(name)]

	if !ok {
		return nil, fmt.Errorf("no attribute %q on NestedAttributeObject", name)
	}

	return attribute, nil
}

// Equal returns true if the given NestedAttributeObject is equivalent.
func (o NestedAttributeObject) Equal(other NestedAttributeObject) bool {
	if !o.Type().Equal(other.Type()) {
		return false
	}

	if len(o.Attributes) != len(other.Attributes) {
		return false
	}

	for name, oAttribute := range o.Attributes {
		otherAttribute, ok := other.Attributes[name]

		if !ok {
			return false
		}

		if !oAttribute.Equal(otherAttribute) {
			return false
		}
	}

	return true
}

// Type returns the framework type of the NestedAttributeObject.
func (o NestedAttributeObject) Type() attr.Type {
	if o.CustomType != nil {
		return o.CustomType
	}

	attrTypes := make(map[string]attr.Type, len(o.Attributes))

	for name, attribute := range o.Attributes {
		attrTypes[name] = attribute.GetType()
	}

	return types.ObjectType{
		AttrTypes: attrTypes,
	}
}
