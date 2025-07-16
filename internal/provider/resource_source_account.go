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

var _ resource.Resource = &SourceAccountResource{}
var _ resource.ResourceWithImportState = &SourceAccountResource{}

func NewSourceAccountResource() resource.Resource {
	return &SourceAccountResource{}
}

type SourceAccountResource struct {
	client *client.EonClient
}

type SourceAccountResourceModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ProviderAccountId types.String `tfsdk:"provider_account_id"`
	CloudProvider     types.String `tfsdk:"cloud_provider"`
	Role              types.String `tfsdk:"role"`
	Status            types.String `tfsdk:"status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *SourceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_account"
}

func (r *SourceAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Connects a source cloud account to the Eon project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Eon-assigned account ID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Account display name in Eon.",
				Required:            true,
			},
			"provider_account_id": schema.StringAttribute{
				MarkdownDescription: "Cloud-provider-assigned account ID.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "Cloud provider. Possible values: `AWS`, `AZURE`, `GCP`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "ARN of the role Eon assumes to access the account in AWS.",
				Optional:            true,
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Connection status of the AWS account, Azure subscription, or GCP project. Only `CONNECTED` source accounts can be backed up. Possible values: `CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the source account was connected to the Eon project.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the source account was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *SourceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SourceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate role is provided for new account creation
	if data.Role.IsNull() || data.Role.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Role",
			"The 'role' attribute is required when creating a new source account. Please provide the ARN of the IAM role that Eon should assume.",
		)
		return
	}

	if data.CloudProvider.ValueString() != string(externalEonSdkAPI.AWS) {
		resp.Diagnostics.AddError(
			"Unsupported Provider",
			"Currently only AWS accounts are supported for account creation",
		)
		return
	}

	config := externalEonSdkAPI.NewAccountConfigInput(externalEonSdkAPI.AWS)
	awsConfig := externalEonSdkAPI.NewAwsAccountConfigInput(data.Role.ValueString())

	config.SetAws(*awsConfig)

	connectReq := externalEonSdkAPI.ConnectSourceAccountRequest{
		Name:                    data.Name.ValueString(),
		SourceAccountAttributes: *config,
	}

	tflog.Debug(ctx, "Connecting source account", map[string]interface{}{
		"name":     data.Name.ValueString(),
		"provider": data.CloudProvider.ValueString(),
		"role":     data.Role.ValueString(),
	})

	// Connect the source account
	account, err := r.client.ConnectSourceAccount(ctx, connectReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect source account: %s", err))
		return
	}

	data.Id = types.StringValue(account.Id)
	data.Status = types.StringValue(string(account.Status))
	data.Name = types.StringValue(account.GetName())
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

	if account.SourceAccountAttributes.HasCloudProvider() {
		data.CloudProvider = types.StringValue(string(account.SourceAccountAttributes.GetCloudProvider()))
	} else {
		data.CloudProvider = types.StringValue(data.CloudProvider.ValueString())
	}

	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Debug(ctx, "Source account connected", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
		"status": data.Status.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accounts, err := r.client.ListSourceAccounts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source accounts: %s", err))
		return
	}

	var found bool
	for _, account := range accounts {
		if account.Id == data.Id.ValueString() {
			found = true
			data.Name = types.StringValue(account.GetName())
			data.Status = types.StringValue(string(account.Status))
			data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

			if account.SourceAccountAttributes.HasCloudProvider() {
				data.CloudProvider = types.StringValue(string(account.SourceAccountAttributes.GetCloudProvider()))
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

func (r *SourceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SourceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("Update Not Supported", "Most source account changes require replacement. Please update your configuration to force replacement if needed.")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Disconnecting source account", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	err := r.client.DisconnectSourceAccount(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disconnect source account: %s", err))
		return
	}

	tflog.Debug(ctx, "Source account disconnected", map[string]interface{}{
		"id": data.Id.ValueString(),
	})
}

func (r *SourceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
