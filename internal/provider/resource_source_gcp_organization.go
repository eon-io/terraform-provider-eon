package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &SourceGcpOrganizationResource{}
var _ resource.ResourceWithImportState = &SourceGcpOrganizationResource{}

func NewSourceGcpOrganizationResource() resource.Resource {
	return &SourceGcpOrganizationResource{}
}

type SourceGcpOrganizationResource struct {
	client *client.EonClient
}

type SourceGcpOrganizationResourceModel struct {
	Id                         types.String `tfsdk:"id"`
	OrganizationId             types.String `tfsdk:"organization_id"`
	ManagementServiceAccountId types.String `tfsdk:"management_service_account_id"`
	ManagementProjectId        types.String `tfsdk:"management_project_id"`
	Name                       types.String `tfsdk:"name"`
	State                      types.String `tfsdk:"state"`
	ExcludeProjectPatterns     types.List   `tfsdk:"exclude_project_patterns"`
	CreatedAt                  types.String `tfsdk:"created_at"`
	UpdatedAt                  types.String `tfsdk:"updated_at"`
}

func (r *SourceGcpOrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_gcp_organization"
}

func (r *SourceGcpOrganizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Connects a GCP organization to the Eon project. All GCP projects within the organization will be automatically discovered and available for backup. Projects matching the exclusion patterns will be skipped during discovery.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Eon-assigned organization ID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "GCP organization ID.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"management_service_account_id": schema.StringAttribute{
				MarkdownDescription: "Email of the GCP service account used to manage the organization.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"management_project_id": schema.StringAttribute{
				MarkdownDescription: "GCP project ID where the management service account resides. Computed from the service account email.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the GCP organization in Eon.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Connection state of the GCP organization. Possible values: `ACTIVE`, `INACTIVE`.",
				Computed:            true,
			},
			"exclude_project_patterns": schema.ListAttribute{
				MarkdownDescription: "Glob patterns for GCP project IDs to exclude from discovery (e.g., `[\"internal-*\", \"test-*\"]`).",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the GCP organization was connected to the Eon project.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the GCP organization was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *SourceGcpOrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SourceGcpOrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceGcpOrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectReq := client.ConnectGcpOrganizationRequest{
		OrganizationId:             data.OrganizationId.ValueString(),
		ManagementServiceAccountId: data.ManagementServiceAccountId.ValueString(),
	}

	if !data.ExcludeProjectPatterns.IsNull() && !data.ExcludeProjectPatterns.IsUnknown() {
		var patterns []string
		resp.Diagnostics.Append(data.ExcludeProjectPatterns.ElementsAs(ctx, &patterns, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		connectReq.ExcludeProjectPatterns = patterns
	}

	tflog.Debug(ctx, "Connecting GCP organization", map[string]interface{}{
		"organization_id":               data.OrganizationId.ValueString(),
		"management_service_account_id": data.ManagementServiceAccountId.ValueString(),
	})

	org, err := r.client.ConnectGcpOrganization(ctx, connectReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect GCP organization: %s", err))
		return
	}

	r.mapOrgToState(ctx, org, &data, &resp.Diagnostics)
	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceGcpOrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceGcpOrganizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgs, err := r.client.ListGcpOrganizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read GCP organizations: %s", err))
		return
	}

	var found bool
	for _, org := range orgs {
		if org.Id == data.Id.ValueString() {
			found = true
			r.mapOrgToState(ctx, &org, &data, &resp.Diagnostics)

			if data.CreatedAt.IsNull() || data.CreatedAt.IsUnknown() {
				data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			}
			data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceGcpOrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SourceGcpOrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("Update Not Supported", "GCP organization changes require replacement.")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceGcpOrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceGcpOrganizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deactivating GCP organization", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	err := r.client.DeactivateGcpOrganization(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to deactivate GCP organization: %s", err))
		return
	}

	tflog.Debug(ctx, "GCP organization deactivated", map[string]interface{}{
		"id": data.Id.ValueString(),
	})
}

func (r *SourceGcpOrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	orgs, err := r.client.ListGcpOrganizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read GCP organizations during import: %s", err))
		return
	}

	var data SourceGcpOrganizationResourceModel
	var found bool

	for _, org := range orgs {
		if org.Id == req.ID {
			found = true
			r.mapOrgToState(ctx, &org, &data, &resp.Diagnostics)
			data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("Resource Not Found", fmt.Sprintf("GCP organization with ID %s not found", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Successfully imported GCP organization", map[string]interface{}{
		"id":    data.Id.ValueString(),
		"state": data.State.ValueString(),
	})
}

func (r *SourceGcpOrganizationResource) mapOrgToState(ctx context.Context, org *client.GcpOrganization, data *SourceGcpOrganizationResourceModel, diags *diag.Diagnostics) {
	data.Id = types.StringValue(org.Id)
	data.OrganizationId = types.StringValue(org.OrganizationId)
	data.ManagementServiceAccountId = types.StringValue(org.ManagementServiceAccountId)
	data.ManagementProjectId = types.StringValue(org.ManagementProjectId)
	data.Name = types.StringValue(org.Name)
	data.State = types.StringValue(org.State)

	if len(org.ExcludeProjectPatterns) > 0 {
		patternList, d := types.ListValueFrom(ctx, types.StringType, org.ExcludeProjectPatterns)
		diags.Append(d...)
		data.ExcludeProjectPatterns = patternList
	} else if !data.ExcludeProjectPatterns.IsNull() {
		patternList, d := types.ListValueFrom(ctx, types.StringType, []string{})
		diags.Append(d...)
		data.ExcludeProjectPatterns = patternList
	} else {
		data.ExcludeProjectPatterns = types.ListNull(types.StringType)
	}
}
