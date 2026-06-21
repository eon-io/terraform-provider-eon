package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &RestoreAwsOrganizationalUnitResource{}
var _ resource.ResourceWithImportState = &RestoreAwsOrganizationalUnitResource{}

func NewRestoreAwsOrganizationalUnitResource() resource.Resource {
	return &RestoreAwsOrganizationalUnitResource{}
}

type RestoreAwsOrganizationalUnitResource struct {
	client *client.EonClient
}

type RestoreAwsOrganizationalUnitResourceModel struct {
	Id                           types.String `tfsdk:"id"`
	Name                         types.String `tfsdk:"name"`
	RoleArn                      types.String `tfsdk:"role_arn"`
	ProviderOrganizationalUnitId types.String `tfsdk:"provider_organizational_unit_id"`
	ProviderManagementAccountId  types.String `tfsdk:"provider_management_account_id"`
	Status                       types.String `tfsdk:"status"`
	CreatedAt                    types.String `tfsdk:"created_at"`
	UpdatedAt                    types.String `tfsdk:"updated_at"`
}

func (r *RestoreAwsOrganizationalUnitResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_aws_organizational_unit"
}

func (r *RestoreAwsOrganizationalUnitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Connects a restore AWS organizational unit to the Eon project as a restore target. All current and future descendant AWS accounts within the organizational unit are discovered and available as restore accounts.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Eon-assigned organizational unit ID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The organizational unit display name in Eon.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"role_arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the role Eon assumes to access the organizational unit in AWS.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"provider_organizational_unit_id": schema.StringAttribute{
				MarkdownDescription: "AWS Organizational Unit ID.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"provider_management_account_id": schema.StringAttribute{
				MarkdownDescription: "AWS Organization management account ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Connection status of the AWS organizational unit. Possible values: `CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the restore AWS organizational unit was connected to the Eon project.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the restore AWS organizational unit was last updated.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *RestoreAwsOrganizationalUnitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.EonClient, got: %T", req.ProviderData))
		return
	}

	r.client = c
}

func (r *RestoreAwsOrganizationalUnitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RestoreAwsOrganizationalUnitResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectReq := client.ConnectRestoreAwsOrganizationalUnitRequest{
		RoleArn:                      data.RoleArn.ValueString(),
		ProviderOrganizationalUnitId: data.ProviderOrganizationalUnitId.ValueString(),
	}

	tflog.Debug(ctx, "Connecting restore AWS organizational unit", map[string]interface{}{
		"role_arn":                        data.RoleArn.ValueString(),
		"provider_organizational_unit_id": data.ProviderOrganizationalUnitId.ValueString(),
	})

	ou, err := r.client.ConnectRestoreAwsOrganizationalUnit(ctx, connectReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect restore AWS organizational unit: %s", err))
		return
	}

	data.Id = types.StringValue(ou.Id)
	data.Name = types.StringValue(ou.Name)
	data.RoleArn = types.StringValue(ou.RoleArn)
	data.ProviderOrganizationalUnitId = types.StringValue(ou.ProviderOrganizationalUnitId)
	data.ProviderManagementAccountId = types.StringValue(ou.ProviderManagementAccountId)
	data.Status = types.StringValue(ou.Status)
	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreAwsOrganizationalUnitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RestoreAwsOrganizationalUnitResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ous, err := r.client.ListRestoreAwsOrganizationalUnits(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore AWS organizational units: %s", err))
		return
	}

	var found bool
	for _, ou := range ous {
		if ou.Id == data.Id.ValueString() {
			found = true
			data.Name = types.StringValue(ou.Name)
			data.RoleArn = types.StringValue(ou.RoleArn)
			data.ProviderOrganizationalUnitId = types.StringValue(ou.ProviderOrganizationalUnitId)
			data.ProviderManagementAccountId = types.StringValue(ou.ProviderManagementAccountId)
			data.Status = types.StringValue(ou.Status)

			if data.CreatedAt.IsNull() || data.CreatedAt.IsUnknown() {
				data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			}
			if data.UpdatedAt.IsNull() || data.UpdatedAt.IsUnknown() {
				data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			}

			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreAwsOrganizationalUnitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RestoreAwsOrganizationalUnitResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("Update Not Supported", "Most restore AWS organizational unit changes require replacement. Please update your configuration to force replacement if needed.")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreAwsOrganizationalUnitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RestoreAwsOrganizationalUnitResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Disconnecting restore AWS organizational unit", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	err := r.client.DisconnectRestoreAwsOrganizationalUnit(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disconnect restore AWS organizational unit: %s", err))
		return
	}
}

func (r *RestoreAwsOrganizationalUnitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	ous, err := r.client.ListRestoreAwsOrganizationalUnits(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore AWS organizational units during import: %s", err))
		return
	}

	var found bool
	var data RestoreAwsOrganizationalUnitResourceModel

	for _, ou := range ous {
		if ou.Id == req.ID {
			found = true

			data.Id = types.StringValue(ou.Id)
			data.Name = types.StringValue(ou.Name)
			data.RoleArn = types.StringValue(ou.RoleArn)
			data.ProviderOrganizationalUnitId = types.StringValue(ou.ProviderOrganizationalUnitId)
			data.ProviderManagementAccountId = types.StringValue(ou.ProviderManagementAccountId)
			data.Status = types.StringValue(ou.Status)
			data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Resource Not Found",
			fmt.Sprintf("Restore AWS organizational unit with ID %s not found", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
