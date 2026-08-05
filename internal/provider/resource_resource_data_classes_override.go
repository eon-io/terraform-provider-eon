package provider

import (
	"context"
	"errors"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ResourceDataClassesOverrideResource{}
var _ resource.ResourceWithImportState = &ResourceDataClassesOverrideResource{}

func NewResourceDataClassesOverrideResource() resource.Resource {
	return &ResourceDataClassesOverrideResource{}
}

type ResourceDataClassesOverrideResource struct {
	client *client.EonClient
}

type ResourceDataClassesOverrideResourceModel struct {
	Id          types.String `tfsdk:"id"`
	ResourceId  types.String `tfsdk:"resource_id"`
	DataClasses types.Set    `tfsdk:"data_classes"`
}

func (r *ResourceDataClassesOverrideResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_data_classes_override"
}

func (r *ResourceDataClassesOverrideResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manually overrides the data classes for an inventory resource, replacing auto-classification.\n\n" +
			"When this resource is created or updated, the listed data classes are applied as a manual override. " +
			"When destroyed, the override is removed and auto-classification resumes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource. Mirrors `resource_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource whose data classes are overridden.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"data_classes": schema.SetAttribute{
				MarkdownDescription: "List of data classes to set as the manual override.",
				ElementType:         types.StringType,
				Required:            true,
			},
		},
	}
}

func (r *ResourceDataClassesOverrideResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	eonClient, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.EonClient, got: %T", req.ProviderData))
		return
	}

	r.client = eonClient
}

func (r *ResourceDataClassesOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceDataClassesOverrideResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOverride(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to override data classes: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceDataClassesOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceDataClassesOverrideResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()

	inventoryResource, err := r.client.GetResourceById(ctx, resourceId)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read data classes override: %s", err))
		return
	}

	dataClasses, ok := dataClassesOverrideFromInventoryResource(inventoryResource)
	if !ok {
		tflog.Debug(ctx, "Data classes override is no longer present; removing from state", map[string]interface{}{
			"resource_id": resourceId,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(inventoryResource.GetId())
	data.ResourceId = types.StringValue(inventoryResource.GetId())
	data.DataClasses = stringSliceToSet(dataClasses)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceDataClassesOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceDataClassesOverrideResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOverride(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update data classes override: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceDataClassesOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceDataClassesOverrideResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()

	tflog.Debug(ctx, "Removing data classes override", map[string]interface{}{
		"resource_id": resourceId,
	})

	err := r.client.RemoveDataClassesOverride(ctx, resourceId)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove data classes override: %s", err))
		return
	}

	tflog.Debug(ctx, "Data classes override removed", map[string]interface{}{
		"resource_id": resourceId,
	})
}

func (r *ResourceDataClassesOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), req.ID)...)

	tflog.Info(ctx, "Successfully imported data classes override", map[string]interface{}{
		"resource_id": req.ID,
	})
}

func (r *ResourceDataClassesOverrideResource) applyOverride(ctx context.Context, data *ResourceDataClassesOverrideResourceModel) error {
	resourceId := data.ResourceId.ValueString()
	dataClasses, err := setToStringSlice(ctx, data.DataClasses)
	if err != nil {
		return err
	}

	tflog.Debug(ctx, "Overriding data classes", map[string]interface{}{
		"resource_id":  resourceId,
		"data_classes": dataClasses,
	})

	overridden, err := r.client.OverrideDataClasses(ctx, resourceId, dataClasses)
	if err != nil {
		return err
	}

	data.Id = types.StringValue(resourceId)
	data.DataClasses = stringSliceToSet(overridden)

	tflog.Debug(ctx, "Data classes overridden", map[string]interface{}{
		"resource_id":  resourceId,
		"data_classes": overridden,
	})

	return nil
}

// dataClassesOverrideFromInventoryResource returns the overridden data classes when a manual override is present.
func dataClassesOverrideFromInventoryResource(inventoryResource *externalEonSdkAPI.InventoryResource) ([]string, bool) {
	if inventoryResource == nil || inventoryResource.Classifications == nil {
		return nil, false
	}

	details, ok := inventoryResource.Classifications.GetDataClassesDetailsOk()
	if !ok || details == nil {
		return nil, false
	}
	if !details.GetIsOverridden() {
		return nil, false
	}

	dataClasses := details.GetDataClasses()
	if dataClasses == nil {
		dataClasses = []string{}
	}
	return dataClasses, true
}

func setToStringSlice(ctx context.Context, set types.Set) ([]string, error) {
	if set.IsNull() || set.IsUnknown() {
		return []string{}, nil
	}

	var values []types.String
	diags := set.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, fmt.Errorf("unable to convert data_classes set: %s", diags.Errors())
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.ValueString())
	}
	return out, nil
}

func stringSliceToSet(values []string) types.Set {
	elems := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elems = append(elems, types.StringValue(value))
	}
	return types.SetValueMust(types.StringType, elems)
}
