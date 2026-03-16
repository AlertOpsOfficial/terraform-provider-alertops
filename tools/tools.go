//go:build tools

package tools

import (
	// ensure tfplugindocs is available for `go generate`
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
