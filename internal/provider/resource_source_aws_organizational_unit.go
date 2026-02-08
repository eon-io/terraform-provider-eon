package provider

import (
	"context"
	"fmt"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &SourceAwsOrganizationalUnitResource{}
var _ resource.ResourceWithImportState = &SourceAwsOrganizationalUnitResource{}

func NewSourceAwsOrganizationalUnitResource() resource.Resource {
	return &SourceAwsOrganizationalUnitResource{}
}

type SourceAwsOrganizationalUnitResource struct {
	client *client.EonClient
}

type SourceAwsOrganizationalUnitResourceModel struct {
	Id                           types.String `tfsdk:"id"`
	Name                         types.String `tfsdk:"name"`
	RoleArn                      types.String `tfsdk:"role_arn"`
	ProviderOrganizationalUnitId types.String `tfsdk:"provider_organizational_unit_id"`
	ProviderManagementAccountId  types.String `tfsdk:"provider_management_account_id"`
	Status                       types.String `tfsdk:"status"`
	CreatedAt                    types.String `tfsdk:"created_at"`
	UpdatedAt                    types.String `tfsdk:"updated_at"`
}

func (r *SourceAwsOrganizationalUnitResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_aws_organizational_unit"
}

func (r *SourceAwsOrganizationalUnitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Connects a source AWS organizational unit to the Eon project. All AWS accounts within the organizational unit (and its nested OUs) will be automatically discovered and available for backup.",
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
				MarkdownDescription: "Date and time the source AWS organizational unit was connected to the Eon project.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the source AWS organizational unit was last updated.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *SourceAwsOrganizationalUnitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.EonClient, got: %T", req.ProviderData))
		return
	}

	r.client = client
}

func (r *SourceAwsOrganizationalUnitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceAwsOrganizationalUnitResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	connectReq := externalEonSdkAPI.ConnectSourceAwsOrganizationalUnitRequest{
		RoleArn:                      data.RoleArn.ValueString(),
		ProviderOrganizationalUnitId: data.ProviderOrganizationalUnitId.ValueString(),
	}

	tflog.Debug(ctx, "Connecting source AWS organizational unit", map[string]interface{}{
		"role_arn":                        data.RoleArn.ValueString(),
		"provider_organizational_unit_id": data.ProviderOrganizationalUnitId.ValueString(),
	})

	ou, err := r.client.ConnectSourceAwsOrganizationalUnit(ctx, connectReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect source AWS organizational unit: %s", err))
		return
	}

	data.Id = types.StringValue(ou.GetId())
	data.Name = types.StringValue(ou.GetName())
	data.RoleArn = types.StringValue(ou.GetRoleArn())
	data.ProviderOrganizationalUnitId = types.StringValue(ou.GetProviderOrganizationalUnitId())
	data.ProviderManagementAccountId = types.StringValue(ou.GetProviderManagementAccountId())
	data.Status = types.StringValue(string(ou.GetStatus()))
	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Debug(ctx, "Source AWS organizational unit connected", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
		"status": data.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAwsOrganizationalUnitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceAwsOrganizationalUnitResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ous, err := r.client.ListSourceAwsOrganizationalUnits(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source AWS organizational units: %s", err))
		return
	}

	var found bool
	for _, ou := range ous {
		if ou.GetId() == data.Id.ValueString() {
			found = true
			data.Name = types.StringValue(ou.GetName())
			data.RoleArn = types.StringValue(ou.GetRoleArn())
			data.ProviderOrganizationalUnitId = types.StringValue(ou.GetProviderOrganizationalUnitId())
			data.ProviderManagementAccountId = types.StringValue(ou.GetProviderManagementAccountId())
			data.Status = types.StringValue(string(ou.GetStatus()))

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

func (r *SourceAwsOrganizationalUnitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SourceAwsOrganizationalUnitResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("Update Not Supported", "Most source AWS organizational unit changes require replacement. Please update your configuration to force replacement if needed.")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAwsOrganizationalUnitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceAwsOrganizationalUnitResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Disconnecting source AWS organizational unit", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	err := r.client.DisconnectSourceAwsOrganizationalUnit(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disconnect source AWS organizational unit: %s", err))
		return
	}

	tflog.Debug(ctx, "Source AWS organizational unit disconnected", map[string]interface{}{
		"id": data.Id.ValueString(),
	})
}

func (r *SourceAwsOrganizationalUnitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	ous, err := r.client.ListSourceAwsOrganizationalUnits(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source AWS organizational units during import: %s", err))
		return
	}

	var found bool
	var data SourceAwsOrganizationalUnitResourceModel

	for _, ou := range ous {
		if ou.GetId() == req.ID {
			found = true

			data.Id = types.StringValue(ou.GetId())
			data.Name = types.StringValue(ou.GetName())
			data.RoleArn = types.StringValue(ou.GetRoleArn())
			data.ProviderOrganizationalUnitId = types.StringValue(ou.GetProviderOrganizationalUnitId())
			data.ProviderManagementAccountId = types.StringValue(ou.GetProviderManagementAccountId())
			data.Status = types.StringValue(string(ou.GetStatus()))
			data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Resource Not Found",
			fmt.Sprintf("Source AWS organizational unit with ID %s not found", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Successfully imported source AWS organizational unit", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
		"status": data.Status.ValueString(),
	})
}
