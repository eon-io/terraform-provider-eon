//go:build tools

package tools

import (
	// Tool dependencies for go generate - not imported in main code
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

