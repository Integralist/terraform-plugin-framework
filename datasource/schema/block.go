package schema

import (
	"github.com/hashicorp/terraform-plugin-framework/internal/fwschema"
	"github.com/hashicorp/terraform-plugin-framework/internal/fwschema/fwxschema"
)

type Block interface {
	fwschema.Block
	fwxschema.BlockWithValidators
}
