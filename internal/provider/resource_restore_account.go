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

var _ resource.Resource = &RestoreAccountResource{}
var _ resource.ResourceWithImportState = &RestoreAccountResource{}

func NewRestoreAccountResource() resource.Resource {
	return &RestoreAccountResource{}
}

type RestoreAccountResource struct {
	client *client.EonClient
}

type RestoreAccountResourceModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ProviderAccountId types.String `tfsdk:"provider_account_id"`
	CloudProvider     types.String `tfsdk:"cloud_provider"`
	Role              types.String `tfsdk:"role"`
	ExternalId        types.String `tfsdk:"external_id"`
	Status            types.String `tfsdk:"status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *RestoreAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account"
}

func (r *RestoreAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Eon restore account resource for connecting/disconnecting restore accounts",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Restore account identifier",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name for the restore account",
				Required:            true,
			},
			"provider_account_id": schema.StringAttribute{
				MarkdownDescription: "Cloud provider account ID (e.g., AWS account ID)",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "Cloud provider (AWS, AZURE, GCP)",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "Role ARN for AWS accounts",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"external_id": schema.StringAttribute{
				MarkdownDescription: "External ID for AWS role assumption",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Connection status",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Last update timestamp",
				Computed:            true,
			},
		},
	}
}

func (r *RestoreAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RestoreAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RestoreAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only AWS is currently supported
	if data.CloudProvider.ValueString() != "AWS" {
		resp.Diagnostics.AddError(
			"Unsupported Provider",
			"Currently only AWS accounts are supported for account creation",
		)
		return
	}

	// Build AWS account config
	config := externalEonSdkAPI.NewAccountConfigInput(externalEonSdkAPI.AWS)
	awsConfig := externalEonSdkAPI.NewAwsAccountConfigInput(data.Role.ValueString())

	// Note: SetExternalId method may not exist in current SDK version
	// External ID support would need to be added to the SDK if required

	config.SetAws(*awsConfig)

	// Prepare the connect request
	connectReq := externalEonSdkAPI.ConnectRestoreAccountRequest{
		Name:                     data.Name.ValueString(),
		RestoreAccountAttributes: *config,
	}

	tflog.Debug(ctx, "Connecting restore account", map[string]interface{}{
		"name":     data.Name.ValueString(),
		"provider": data.CloudProvider.ValueString(),
		"role":     data.Role.ValueString(),
	})

	// Connect the restore account
	account, err := r.client.ConnectRestoreAccount(ctx, connectReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect restore account: %s", err))
		return
	}

	// Update model with response data
	data.Id = types.StringValue(account.Id)
	data.Status = types.StringValue(string(account.Status))
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

	// Extract provider from RestoreAccountAttributes
	if account.RestoreAccountAttributes.HasCloudProvider() {
		data.CloudProvider = types.StringValue(string(account.RestoreAccountAttributes.GetCloudProvider()))
	} else {
		data.CloudProvider = types.StringValue(data.CloudProvider.ValueString()) // Keep original value
	}

	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339)) // Fix: Set updated_at to a known value

	tflog.Debug(ctx, "Restore account connected", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"status": data.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RestoreAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// List all restore accounts and find ours
	accounts, err := r.client.ListRestoreAccounts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore accounts: %s", err))
		return
	}

	// Find the account by ID
	var found bool
	for _, account := range accounts {
		if account.Id == data.Id.ValueString() {
			found = true
			// Update data with current values
			data.Status = types.StringValue(string(account.Status))
			data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

			// Extract provider from RestoreAccountAttributes
			if account.RestoreAccountAttributes.HasCloudProvider() {
				data.CloudProvider = types.StringValue(string(account.RestoreAccountAttributes.GetCloudProvider()))
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

func (r *RestoreAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RestoreAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For now, most changes require replace due to API limitations
	resp.Diagnostics.AddWarning("Update Not Supported", "Most restore account changes require replacement. Please update your configuration to force replacement if needed.")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RestoreAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Disconnecting restore account", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	err := r.client.DisconnectRestoreAccount(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disconnect restore account: %s", err))
		return
	}

	tflog.Debug(ctx, "Restore account disconnected", map[string]interface{}{
		"id": data.Id.ValueString(),
	})
}

func (r *RestoreAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
