package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// The Update paths of these resources stamp updated_at with time.Now(). Pinning the attribute to
// prior state during plan therefore makes every apply return a value the plan did not promise,
// which Terraform rejects as "Provider produced inconsistent result after apply".
func TestUpdatedAtIsNotPinnedToState(t *testing.T) {
	resources := map[string]resource.Resource{
		"eon_source_account":  NewSourceAccountResource(),
		"eon_restore_account": NewRestoreAccountResource(),
	}

	for name, res := range resources {
		t.Run(name, func(t *testing.T) {
			resp := &resource.SchemaResponse{}
			res.Schema(context.Background(), resource.SchemaRequest{}, resp)

			attr, ok := resp.Schema.Attributes["updated_at"]
			if !ok {
				t.Fatalf("%s has no updated_at attribute", name)
			}

			stringAttr, ok := attr.(schema.StringAttribute)
			if !ok {
				t.Fatalf("%s updated_at is %T, want schema.StringAttribute", name, attr)
			}

			if len(stringAttr.PlanModifiers) != 0 {
				t.Errorf("%s updated_at has %d plan modifiers, want none so the plan leaves it unknown",
					name, len(stringAttr.PlanModifiers))
			}
		})
	}
}
