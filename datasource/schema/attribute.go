package schema

import (
	"github.com/hashicorp/terraform-plugin-framework/internal/fwschema"
)

type Attribute interface {
	fwschema.Attribute
	//fwxschema.AttributeWithValidators
}
