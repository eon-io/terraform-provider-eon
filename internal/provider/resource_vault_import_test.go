package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ImportState writes a fresh VaultResourceModel; a Labels set left at its zero value has no element
// type and State.Set rejects it ("MISSING TYPE"), which breaks every import.
func TestVaultResourceModel_StateSetRequiresTypedLabels(t *testing.T) {
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	NewVaultResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	newState := func() *tfsdk.State {
		return &tfsdk.State{Schema: schemaResp.Schema}
	}
	model := func(labels types.Set) VaultResourceModel {
		return VaultResourceModel{
			Id:                types.StringValue("id"),
			Name:              types.StringValue("name"),
			Region:            types.StringValue("us-east-1"),
			CloudProvider:     types.StringValue("AWS"),
			AwsKmsKeyArn:      types.StringNull(),
			VaultAccountId:    types.StringValue("a"),
			ProviderAccountId: types.StringValue("b"),
			IsManagedByEon:    types.BoolValue(false),
			Labels:            labels,
		}
	}

	untyped := model(types.Set{})
	if diags := newState().Set(ctx, &untyped); !diags.HasError() {
		t.Fatal("expected zero-value labels set to be rejected")
	}

	typed := model(labelsToSet(nil))
	if diags := newState().Set(ctx, &typed); diags.HasError() {
		t.Fatalf("typed null labels rejected: %v", diags)
	}
}
