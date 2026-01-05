package provider

import (
	"context"
	"errors"
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
	Id                types.String             `tfsdk:"id"`
	Name              types.String             `tfsdk:"name"`
	ProviderAccountId types.String             `tfsdk:"provider_account_id"` // Deprecated, now computed
	CloudProvider     types.String             `tfsdk:"cloud_provider"`
	Role              types.String             `tfsdk:"role"` // Deprecated, use aws block
	Status            types.String             `tfsdk:"status"`
	CreatedAt         types.String             `tfsdk:"created_at"`
	UpdatedAt         types.String             `tfsdk:"updated_at"`
	Aws               *AwsAccountConfigModel   `tfsdk:"aws"`
	Azure             *AzureAccountConfigModel `tfsdk:"azure"`
	Gcp               *GcpAccountConfigModel   `tfsdk:"gcp"`
}

func (r *RestoreAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account"
}

func (r *RestoreAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Connects a restore account to the Eon project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Eon-assigned restore account ID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Account display name in Eon.",
				Required:            true,
			},
			"provider_account_id": schema.StringAttribute{
				MarkdownDescription: "Cloud-provider-assigned account ID (AWS account ID or Azure subscription ID). Computed from the `aws` or `azure` block.",
				Computed:            true,
				DeprecationMessage:  "This field is now computed from the aws or azure block.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "Cloud provider. Possible values: `AWS`, `AZURE`, `GCP`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "**Deprecated:** Use `aws { role_arn = \"...\" }` instead. ARN of the role Eon assumes to access the account in AWS.",
				Optional:            true,
				Computed:            true,
				DeprecationMessage:  "Use 'aws { role_arn = \"...\" }' instead.",
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Connection status of the AWS account, Azure subscription, or GCP project. Only `CONNECTED` restore accounts can be restored to. Possible values: `CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the restore account was connected to the Eon project.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the restore account was last updated.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			CloudProviderAWS.BlockName():   awsSchemaBlock(),
			CloudProviderAzure.BlockName(): azureSchemaBlock("Scope restores to this resource group. When provided, only resources in this resource group can be restored to."),
			CloudProviderGCP.BlockName():   gcpSchemaBlock(),
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

	cloudProvider := CloudProvider(data.CloudProvider.ValueString())
	config := externalEonSdkAPI.NewRestoreAccountAttributesInput(externalEonSdkAPI.Provider(cloudProvider))

	switch cloudProvider {
	case CloudProviderAWS:
		var roleArn string
		if data.Aws != nil && !data.Aws.RoleArn.IsNull() {
			roleArn = data.Aws.RoleArn.ValueString()
		} else if !data.Role.IsNull() && data.Role.ValueString() != "" {
			// Legacy fallback
			roleArn = data.Role.ValueString()
		} else {
			resp.Diagnostics.AddError(
				"Missing Configuration",
				"Either 'aws { role_arn = \"...\" }' or deprecated 'role' attribute is required for AWS accounts.",
			)
			return
		}
		awsConfig := externalEonSdkAPI.NewAwsRestoreAccountAttributesInput(roleArn)
		config.SetAws(*awsConfig)

		tflog.Debug(ctx, "Connecting AWS restore account", map[string]interface{}{
			"name":     data.Name.ValueString(),
			"role_arn": roleArn,
		})

	case CloudProviderAzure:
		if data.Azure == nil {
			resp.Diagnostics.AddError(
				"Missing Configuration",
				"The 'azure' block is required when cloud_provider is AZURE.",
			)
			return
		}
		if data.Azure.TenantId.IsNull() || data.Azure.SubscriptionId.IsNull() {
			resp.Diagnostics.AddError(
				"Missing Configuration",
				"Both 'tenant_id' and 'subscription_id' are required in the azure block.",
			)
			return
		}
		azureConfig := externalEonSdkAPI.NewAzureRestoreAccountAttributesInput(
			data.Azure.TenantId.ValueString(),
			data.Azure.SubscriptionId.ValueString(),
		)
		if !data.Azure.ResourceGroupName.IsNull() && data.Azure.ResourceGroupName.ValueString() != "" {
			azureConfig.SetResourceGroupName(data.Azure.ResourceGroupName.ValueString())
		}
		config.SetAzure(*azureConfig)

		tflog.Debug(ctx, "Connecting Azure restore account", map[string]interface{}{
			"name":            data.Name.ValueString(),
			"tenant_id":       data.Azure.TenantId.ValueString(),
			"subscription_id": data.Azure.SubscriptionId.ValueString(),
		})

	case CloudProviderGCP:
		if data.Gcp == nil {
			resp.Diagnostics.AddError(
				"Missing Configuration",
				"The 'gcp' block is required when cloud_provider is GCP.",
			)
			return
		}
		if data.Gcp.ProjectId.IsNull() || data.Gcp.ServiceAccount.IsNull() {
			resp.Diagnostics.AddError(
				"Missing Configuration",
				"Both 'project_id' and 'service_account' are required in the gcp block.",
			)
			return
		}
		gcpConfig := externalEonSdkAPI.NewGcpRestoreAccountAttributesInput(data.Gcp.ServiceAccount.ValueString())
		config.SetGcp(*gcpConfig)

		tflog.Debug(ctx, "Connecting GCP restore account", map[string]interface{}{
			"name":            data.Name.ValueString(),
			"project_id":      data.Gcp.ProjectId.ValueString(),
			"service_account": data.Gcp.ServiceAccount.ValueString(),
		})

	default:
		resp.Diagnostics.AddError(
			"Unsupported Provider",
			fmt.Sprintf("Cloud provider '%s' is not supported. Supported values: AWS, AZURE, GCP.", cloudProvider),
		)
		return
	}

	connectReq := externalEonSdkAPI.ConnectRestoreAccountRequest{
		Name:                     data.Name.ValueStringPointer(),
		RestoreAccountAttributes: *config,
	}

	// Connect the restore account
	account, err := r.client.ConnectRestoreAccount(ctx, connectReq)
	if err != nil {
		// Check if this is a 409 Conflict (account already exists)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 409 {
			existingID := r.findExistingAccountID(ctx, cloudProvider, data)
			title, detail := conflictErrorMessage("Restore Account", existingID)
			resp.Diagnostics.AddError(title, fmt.Sprintf("%s\n\nOriginal error: %s", detail, err.Error()))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect restore account: %s", err))
		return
	}

	// Update state from response
	data.Id = types.StringValue(account.Id)
	data.Status = types.StringValue(string(account.Status))
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

	if account.RestoreAccountAttributes.HasCloudProvider() {
		data.CloudProvider = types.StringValue(string(account.RestoreAccountAttributes.GetCloudProvider()))
	}

	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	// Set role to null by default, override for AWS (backward compatibility)
	data.Role = types.StringNull()
	if cloudProvider == CloudProviderAWS && data.Aws != nil {
		data.Role = data.Aws.RoleArn
	}

	tflog.Debug(ctx, "Restore account connected", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
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

	accounts, err := r.client.ListRestoreAccounts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore accounts: %s", err))
		return
	}

	var found bool
	for _, account := range accounts {
		if account.Id == data.Id.ValueString() {
			found = true
			data.Name = types.StringValue(account.GetName())
			data.Status = types.StringValue(string(account.Status))
			data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

			var cloudProvider CloudProvider
			if account.RestoreAccountAttributes.HasCloudProvider() {
				cloudProvider = CloudProvider(account.RestoreAccountAttributes.GetCloudProvider())
				data.CloudProvider = types.StringValue(cloudProvider.String())
			}

			// Populate cloud-specific blocks from API response
			switch cloudProvider {
			case CloudProviderAWS:
				if account.RestoreAccountAttributes.HasAws() {
					awsAttrs := account.RestoreAccountAttributes.GetAws()
					data.Aws = &AwsAccountConfigModel{
						RoleArn: types.StringValue(awsAttrs.GetRoleArn()),
					}
					// Also populate deprecated role field for backward compatibility
					data.Role = types.StringValue(awsAttrs.GetRoleArn())
				}
				data.Azure = nil
				data.Gcp = nil
			case CloudProviderAzure:
				if account.RestoreAccountAttributes.HasAzure() {
					azureAttrs := account.RestoreAccountAttributes.GetAzure()
					data.Azure = &AzureAccountConfigModel{
						TenantId:       types.StringValue(azureAttrs.GetTenantId()),
						SubscriptionId: types.StringValue(account.GetProviderAccountId()),
					}
					if azureAttrs.HasResourceGroupName() {
						data.Azure.ResourceGroupName = types.StringValue(azureAttrs.GetResourceGroupName())
					}
				}
				data.Aws = nil
				data.Gcp = nil
				data.Role = types.StringNull()
			case CloudProviderGCP:
				if account.RestoreAccountAttributes.HasGcp() {
					gcpAttrs := account.RestoreAccountAttributes.GetGcp()
					data.Gcp = &GcpAccountConfigModel{
						ProjectId:      types.StringValue(account.GetProviderAccountId()),
						ServiceAccount: types.StringValue(gcpAttrs.GetServiceAccount()),
					}
				}
				data.Aws = nil
				data.Azure = nil
				data.Role = types.StringNull()
			}

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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	accounts, err := r.client.ListRestoreAccounts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore accounts during import: %s", err))
		return
	}

	var found bool
	var data RestoreAccountResourceModel

	for _, account := range accounts {
		if account.Id == req.ID {
			found = true
			data.Id = types.StringValue(account.Id)
			data.Name = types.StringValue(account.GetName())
			data.Status = types.StringValue(string(account.Status))
			data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

			var cloudProvider CloudProvider
			if account.RestoreAccountAttributes.HasCloudProvider() {
				cloudProvider = CloudProvider(account.RestoreAccountAttributes.GetCloudProvider())
				data.CloudProvider = types.StringValue(cloudProvider.String())
			}

			// Populate cloud-specific blocks from API response
			switch cloudProvider {
			case CloudProviderAWS:
				if account.RestoreAccountAttributes.HasAws() {
					awsAttrs := account.RestoreAccountAttributes.GetAws()
					data.Aws = &AwsAccountConfigModel{
						RoleArn: types.StringValue(awsAttrs.GetRoleArn()),
					}
					// Also populate deprecated role field for backward compatibility
					data.Role = types.StringValue(awsAttrs.GetRoleArn())
				}
			case CloudProviderAzure:
				if account.RestoreAccountAttributes.HasAzure() {
					azureAttrs := account.RestoreAccountAttributes.GetAzure()
					data.Azure = &AzureAccountConfigModel{
						TenantId:       types.StringValue(azureAttrs.GetTenantId()),
						SubscriptionId: types.StringValue(account.GetProviderAccountId()),
					}
					if azureAttrs.HasResourceGroupName() {
						data.Azure.ResourceGroupName = types.StringValue(azureAttrs.GetResourceGroupName())
					}
				}
			case CloudProviderGCP:
				if account.RestoreAccountAttributes.HasGcp() {
					gcpAttrs := account.RestoreAccountAttributes.GetGcp()
					data.Gcp = &GcpAccountConfigModel{
						ProjectId:      types.StringValue(account.GetProviderAccountId()),
						ServiceAccount: types.StringValue(gcpAttrs.GetServiceAccount()),
					}
				}
			}

			data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
			data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Resource Not Found",
			fmt.Sprintf("Restore account with ID %s not found", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Successfully imported restore account", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
		"status": data.Status.ValueString(),
	})
}

// findExistingAccountID attempts to find the ID of an existing restore account
// that matches the given configuration. Returns empty string if not found.
func (r *RestoreAccountResource) findExistingAccountID(ctx context.Context, cloudProvider CloudProvider, data RestoreAccountResourceModel) string {
	accounts, err := r.client.ListRestoreAccounts(ctx)
	if err != nil {
		tflog.Debug(ctx, "Failed to list restore accounts to find existing ID", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	for _, account := range accounts {
		if CloudProvider(account.RestoreAccountAttributes.GetCloudProvider()) != cloudProvider {
			continue
		}

		switch cloudProvider {
		case CloudProviderAWS:
			if data.Aws != nil && account.RestoreAccountAttributes.HasAws() {
				awsAttrs := account.RestoreAccountAttributes.GetAws()
				if awsAttrs.GetRoleArn() == data.Aws.RoleArn.ValueString() {
					return account.Id
				}
			}
		case CloudProviderAzure:
			if data.Azure != nil && account.RestoreAccountAttributes.HasAzure() {
				// Match by subscription_id (the unique identifier for Azure accounts)
				if account.GetProviderAccountId() == data.Azure.SubscriptionId.ValueString() {
					return account.Id
				}
			}
		case CloudProviderGCP:
			if data.Gcp != nil && account.RestoreAccountAttributes.HasGcp() {
				// Match by project_id (the unique identifier for GCP accounts)
				if account.GetProviderAccountId() == data.Gcp.ProjectId.ValueString() {
					return account.Id
				}
			}
		}
	}

	return ""
}
