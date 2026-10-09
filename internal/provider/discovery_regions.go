package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const discoveryRegionsDescription = "Regions Eon discovers resources in. An empty set means every supported region. " +
	"Omit the attribute to leave the regions to the Eon console instead of Terraform. " +
	"Resources in a region removed from the set are treated as deleted from the cloud. " +
	"Discovery honors the regions only when the discovery-regions feature is enabled for the project."

// discoveryRegionsAttribute is Optional+Computed so that omitting it leaves regions managed outside Terraform,
// while an explicit empty set still means "every region"; null and empty differ in the API the same way.
func discoveryRegionsAttribute(description string, extraModifiers ...planmodifier.Set) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		Computed:            true,
		MarkdownDescription: description,
		PlanModifiers:       append([]planmodifier.Set{setplanmodifier.UseStateForUnknown()}, extraModifiers...),
	}
}

// regionsFromSet returns the configured regions, or ok=false when the value is null or unknown (not managed here).
// The result is never nil when ok, because the SDK only serializes a non-nil empty slice as "every region".
func regionsFromSet(ctx context.Context, set types.Set) (regions []string, ok bool, diags diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, false, nil
	}
	regions = []string{}
	diags = set.ElementsAs(ctx, &regions, false)
	slices.Sort(regions)
	return regions, true, diags
}

// regionsToSet always returns a known set: the API reports "every region" as an empty list, and a null here would
// read as drift against a configured `regions = []`.
func regionsToSet(regions []string) types.Set {
	elements := make([]attr.Value, 0, len(regions))
	for _, region := range regions {
		elements = append(elements, types.StringValue(region))
	}
	return types.SetValueMust(types.StringType, elements)
}

// regionsChanged reports whether the plan sets regions that differ from the state.
func regionsChanged(plan, state types.Set) bool {
	if plan.IsNull() || plan.IsUnknown() {
		return false
	}
	return !plan.Equal(state)
}

// immutableRegionsModifier fails the plan when regions change on an existing resource whose API cannot update
// them, instead of replacing it: recreating a connected organizational unit disconnects every member account.
type immutableRegionsModifier struct {
	resourceName string
}

func (m immutableRegionsModifier) Description(context.Context) string {
	return fmt.Sprintf("Fails the plan when regions change on an existing %s.", m.resourceName)
}

func (m immutableRegionsModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m immutableRegionsModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if !regionsChanged(req.PlanValue, req.StateValue) {
		return
	}
	current, _, diags := regionsFromSet(ctx, req.StateValue)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.AddAttributeError(
		path.Root("regions"),
		"Regions Cannot Be Changed",
		fmt.Sprintf("The regions of a connected %s cannot be changed. Set `regions` back to its current value [%s], "+
			"or remove the attribute to stop managing it, or replace the resource with `terraform apply -replace`.",
			m.resourceName, strings.Join(current, ", ")),
	)
}
