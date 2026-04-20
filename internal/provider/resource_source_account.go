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

var _ resource.Resource = &SourceAccountResource{}
var _ resource.ResourceWithImportState = &SourceAccountResource{}

func NewSourceAccountResource() resource.Resource {
	return &SourceAccountResource{}
}

type SourceAccountResource struct {
	client *client.EonClient
}

type SourceAccountResourceModel struct {
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
				MarkdownDescription: "Connection status of the source account. The provider automatically reconnects accounts that drift to `DISCONNECTED`. Possible values: `CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{ReconnectOnDisconnected()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the source account was connected to the Eon project.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the source account was last updated.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			CloudProviderAWS.BlockName():   awsSchemaBlock(),
			CloudProviderAzure.BlockName(): azureSchemaBlock("Scope discovery to this resource group. When provided, only resources in this resource group are discovered."),
			CloudProviderGCP.BlockName():   gcpSchemaBlock(),
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

	cloudProvider := CloudProvider(data.CloudProvider.ValueString())
	config := externalEonSdkAPI.NewSourceAccountAttributesInput(externalEonSdkAPI.Provider(cloudProvider))

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
		awsConfig := externalEonSdkAPI.NewAwsSourceAccountAttributesInput(roleArn)
		config.SetAws(*awsConfig)

		tflog.Debug(ctx, "Connecting AWS source account", map[string]interface{}{
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
		azureConfig := externalEonSdkAPI.NewAzureSourceAccountAttributesInput(
			data.Azure.TenantId.ValueString(),
			data.Azure.SubscriptionId.ValueString(),
		)
		if !data.Azure.ResourceGroupName.IsNull() && data.Azure.ResourceGroupName.ValueString() != "" {
			azureConfig.SetEonInternalResourceGroupName(data.Azure.ResourceGroupName.ValueString())
		}
		config.SetAzure(*azureConfig)

		tflog.Debug(ctx, "Connecting Azure source account", map[string]interface{}{
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
		gcpConfig := externalEonSdkAPI.NewGcpSourceAccountAttributesInput(data.Gcp.ServiceAccount.ValueString())
		config.SetGcp(*gcpConfig)

		tflog.Debug(ctx, "Connecting GCP source account", map[string]interface{}{
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

	connectReq := externalEonSdkAPI.ConnectSourceAccountRequest{
		Name:                    data.Name.ValueStringPointer(),
		SourceAccountAttributes: *config,
	}

	// Connect the source account
	account, err := r.client.ConnectSourceAccount(ctx, connectReq)
	if err != nil {
		// Check if this is a 409 Conflict (account already exists)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 409 {
			// Treat 409 as success — adopt the existing account into state
			existing := r.findExistingAccount(ctx, cloudProvider, data)
			if existing == nil {
				resp.Diagnostics.AddError("Source Account Already Exists",
					fmt.Sprintf("A source account with this configuration already exists but could not be found via the API.\n\nOriginal error: %s", err.Error()))
				return
			}
			tflog.Info(ctx, "Source account already exists (409 Conflict), adopting into state", map[string]interface{}{
				"id":   existing.Id,
				"name": existing.GetName(),
			})
			account = existing
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect source account: %s", err))
			return
		}
	}

	// Update state from response
	data.Id = types.StringValue(account.Id)
	data.Status = types.StringValue(string(account.Status))
	data.Name = types.StringValue(account.GetName())
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())
	data.CloudProvider = types.StringValue(string(account.SourceAccountAttributes.GetCloudProvider()))
	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	// Set role to null by default, override for AWS (backward compatibility)
	data.Role = types.StringNull()
	if cloudProvider == CloudProviderAWS && data.Aws != nil {
		data.Role = data.Aws.RoleArn
	}

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

	account, err := r.client.GetSourceAccount(ctx, data.Id.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source account: %s", err))
		return
	}

	data.Name = types.StringValue(account.GetName())
	data.Status = types.StringValue(string(account.Status))
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

	cloudProvider := CloudProvider(account.SourceAccountAttributes.GetCloudProvider())
	data.CloudProvider = types.StringValue(cloudProvider.String())

	// Populate cloud-specific blocks from API response
	switch cloudProvider {
	case CloudProviderAWS:
		if account.SourceAccountAttributes.HasAws() {
			awsAttrs := account.SourceAccountAttributes.GetAws()
			data.Aws = &AwsAccountConfigModel{
				RoleArn: types.StringValue(awsAttrs.GetRoleArn()),
			}
			// Also populate deprecated role field for backward compatibility
			data.Role = types.StringValue(awsAttrs.GetRoleArn())
		}
		data.Azure = nil
		data.Gcp = nil
	case CloudProviderAzure:
		if account.SourceAccountAttributes.HasAzure() {
			azureAttrs := account.SourceAccountAttributes.GetAzure()
			data.Azure = &AzureAccountConfigModel{
				TenantId:       types.StringValue(azureAttrs.GetTenantId()),
				SubscriptionId: types.StringValue(account.GetProviderAccountId()), // subscription_id is the provider_account_id
			}
			// ResourceGroupName is not returned in the output model for source accounts
		}
		data.Aws = nil
		data.Gcp = nil
		data.Role = types.StringNull()
	case CloudProviderGCP:
		if account.SourceAccountAttributes.HasGcp() {
			gcpAttrs := account.SourceAccountAttributes.GetGcp()
			data.Gcp = &GcpAccountConfigModel{
				ProjectId:      types.StringValue(account.GetProviderAccountId()), // project_id is the provider_account_id
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

	// Surface account statuses the provider cannot auto-remediate so the user
	// sees them in plan output. DISCONNECTED is handled by the plan modifier
	// and the Update flow; anything else non-CONNECTED needs manual attention.
	status := data.Status.ValueString()
	if status != "" && status != "CONNECTED" && status != "DISCONNECTED" {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("status"),
			"Source Account Requires Manual Intervention",
			fmt.Sprintf(
				"Source account %s is in status %q. The provider cannot automatically remediate this state; resolve the underlying issue in the Eon console or cloud provider and re-run.",
				data.Id.ValueString(), status,
			),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SourceAccountResourceModel
	var state SourceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountId := state.Id.ValueString()
	var latestAccount *externalEonSdkAPI.SourceAccount

	// Step 1: Update mutable fields (name, role_arn) if changed.
	updateReq := r.buildUpdateRequest(plan, state)
	if updateReq != nil {
		tflog.Info(ctx, "Updating source account", map[string]interface{}{
			"id": accountId,
		})

		account, err := r.client.UpdateSourceAccount(ctx, accountId, *updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Update Failed",
				fmt.Sprintf("Unable to update source account %s: %s", accountId, err),
			)
			return
		}
		latestAccount = account
	}

	// Step 2: Reconnect if the account is DISCONNECTED.
	if state.Status.ValueString() == "DISCONNECTED" {
		tflog.Info(ctx, "Source account is disconnected, attempting reconnect", map[string]interface{}{
			"id": accountId,
		})

		account, err := r.client.ReconnectSourceAccount(ctx, accountId)
		if err != nil {
			resp.Diagnostics.AddError(
				"Reconnect Failed",
				fmt.Sprintf("Unable to reconnect source account %s: %s", accountId, err),
			)
			return
		}
		latestAccount = account

		tflog.Info(ctx, "Source account reconnected", map[string]interface{}{
			"id": accountId,
		})
	}

	// Step 3: Build new state from the API response, or read back if no response was returned.
	if latestAccount == nil {
		fetched, err := r.readSourceAccount(ctx, accountId, plan)
		if err != nil {
			resp.Diagnostics.AddError(
				"Read After Update Failed",
				fmt.Sprintf("Unable to read source account %s after update: %s", accountId, err),
			)
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, fetched)...)
		return
	}

	newState := r.mapAccountToState(latestAccount, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// buildUpdateRequest compares plan and state and returns an UpdateSourceAccountRequest
// if any mutable fields changed. Returns nil if nothing needs updating.
func (r *SourceAccountResource) buildUpdateRequest(plan, state SourceAccountResourceModel) *client.UpdateSourceAccountRequest {
	var req client.UpdateSourceAccountRequest
	var changed bool

	if plan.Name.ValueString() != state.Name.ValueString() {
		name := plan.Name.ValueString()
		req.Name = &name
		changed = true
	}

	if plan.Aws != nil && state.Aws != nil &&
		plan.Aws.RoleArn.ValueString() != state.Aws.RoleArn.ValueString() {
		roleArn := plan.Aws.RoleArn.ValueString()
		req.SourceAccountAttributes = &client.UpdateSourceAccountAttributes{
			Aws: &client.UpdateAwsSourceAccountAttributes{
				RoleArn: &roleArn,
			},
		}
		changed = true
	}

	if !changed {
		return nil
	}
	return &req
}

// mapAccountToState maps a SourceAccount API response to the Terraform resource model.
// The plan is used to preserve fields not returned by the API (timestamps).
func (r *SourceAccountResource) mapAccountToState(account *externalEonSdkAPI.SourceAccount, plan SourceAccountResourceModel) *SourceAccountResourceModel {
	data := SourceAccountResourceModel{
		Id:                types.StringValue(account.Id),
		Name:              types.StringValue(account.GetName()),
		Status:            types.StringValue(string(account.Status)),
		ProviderAccountId: types.StringValue(account.GetProviderAccountId()),
		CreatedAt:         plan.CreatedAt,
		UpdatedAt:         types.StringValue(time.Now().Format(time.RFC3339)),
	}

	cloudProvider := CloudProvider(account.SourceAccountAttributes.GetCloudProvider())
	data.CloudProvider = types.StringValue(cloudProvider.String())

	switch cloudProvider {
	case CloudProviderAWS:
		if account.SourceAccountAttributes.HasAws() {
			awsAttrs := account.SourceAccountAttributes.GetAws()
			data.Aws = &AwsAccountConfigModel{
				RoleArn: types.StringValue(awsAttrs.GetRoleArn()),
			}
			data.Role = types.StringValue(awsAttrs.GetRoleArn())
		}
	case CloudProviderAzure:
		if account.SourceAccountAttributes.HasAzure() {
			azureAttrs := account.SourceAccountAttributes.GetAzure()
			data.Azure = &AzureAccountConfigModel{
				TenantId:       types.StringValue(azureAttrs.GetTenantId()),
				SubscriptionId: types.StringValue(account.GetProviderAccountId()),
			}
		}
		data.Role = types.StringNull()
	case CloudProviderGCP:
		if account.SourceAccountAttributes.HasGcp() {
			gcpAttrs := account.SourceAccountAttributes.GetGcp()
			data.Gcp = &GcpAccountConfigModel{
				ProjectId:      types.StringValue(account.GetProviderAccountId()),
				ServiceAccount: types.StringValue(gcpAttrs.GetServiceAccount()),
			}
		}
		data.Role = types.StringNull()
	}

	return &data
}

// readSourceAccount fetches a source account by ID and maps it to the resource model.
func (r *SourceAccountResource) readSourceAccount(ctx context.Context, accountId string, plan SourceAccountResourceModel) (*SourceAccountResourceModel, error) {
	account, err := r.client.GetSourceAccount(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("unable to get source account: %w", err)
	}
	return r.mapAccountToState(account, plan), nil
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	account, err := r.client.GetSourceAccount(ctx, req.ID)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.Diagnostics.AddError(
				"Resource Not Found",
				fmt.Sprintf("Source account with ID %s not found", req.ID),
			)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source account during import: %s", err))
		return
	}

	var data SourceAccountResourceModel
	data.Id = types.StringValue(account.Id)
	data.Name = types.StringValue(account.GetName())
	data.Status = types.StringValue(string(account.Status))
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

	cloudProvider := CloudProvider(account.SourceAccountAttributes.GetCloudProvider())
	data.CloudProvider = types.StringValue(cloudProvider.String())

	// Populate cloud-specific blocks from API response
	switch cloudProvider {
	case CloudProviderAWS:
		if account.SourceAccountAttributes.HasAws() {
			awsAttrs := account.SourceAccountAttributes.GetAws()
			data.Aws = &AwsAccountConfigModel{
				RoleArn: types.StringValue(awsAttrs.GetRoleArn()),
			}
			// Also populate deprecated role field for backward compatibility
			data.Role = types.StringValue(awsAttrs.GetRoleArn())
		}
	case CloudProviderAzure:
		if account.SourceAccountAttributes.HasAzure() {
			azureAttrs := account.SourceAccountAttributes.GetAzure()
			data.Azure = &AzureAccountConfigModel{
				TenantId:       types.StringValue(azureAttrs.GetTenantId()),
				SubscriptionId: types.StringValue(account.GetProviderAccountId()),
			}
		}
	case CloudProviderGCP:
		if account.SourceAccountAttributes.HasGcp() {
			gcpAttrs := account.SourceAccountAttributes.GetGcp()
			data.Gcp = &GcpAccountConfigModel{
				ProjectId:      types.StringValue(account.GetProviderAccountId()),
				ServiceAccount: types.StringValue(gcpAttrs.GetServiceAccount()),
			}
		}
	}

	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Successfully imported source account", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
		"status": data.Status.ValueString(),
	})
}

// findExistingAccount attempts to find an existing source account
// that matches the given configuration. Returns nil if not found.
func (r *SourceAccountResource) findExistingAccount(ctx context.Context, cloudProvider CloudProvider, data SourceAccountResourceModel) *externalEonSdkAPI.SourceAccount {
	accounts, err := r.client.ListSourceAccounts(ctx)
	if err != nil {
		tflog.Debug(ctx, "Failed to list source accounts to find existing account", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}

	for _, account := range accounts {
		if CloudProvider(account.SourceAccountAttributes.GetCloudProvider()) != cloudProvider {
			continue
		}

		switch cloudProvider {
		case CloudProviderAWS:
			if data.Aws != nil && account.SourceAccountAttributes.HasAws() {
				awsAttrs := account.SourceAccountAttributes.GetAws()
				if awsAttrs.GetRoleArn() == data.Aws.RoleArn.ValueString() {
					return &account
				}
			}
		case CloudProviderAzure:
			if data.Azure != nil && account.SourceAccountAttributes.HasAzure() {
				if account.GetProviderAccountId() == data.Azure.SubscriptionId.ValueString() {
					return &account
				}
			}
		case CloudProviderGCP:
			if data.Gcp != nil && account.SourceAccountAttributes.HasGcp() {
				if account.GetProviderAccountId() == data.Gcp.ProjectId.ValueString() {
					return &account
				}
			}
		}
	}

	return nil
}
