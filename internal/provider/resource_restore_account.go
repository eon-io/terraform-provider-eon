package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
				MarkdownDescription: "Connection status of the AWS account, Azure subscription, or GCP project. Only `CONNECTED` restore accounts can be restored to. The provider automatically reconnects accounts that drift to `DISCONNECTED`. Possible values: `CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{ReconnectOnDisconnected()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the restore account was connected to the Eon project.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Time at which Terraform last applied a change to this restore account. Eon does not report an update timestamp, so this records the local apply time and is not refreshed afterwards.",
				Computed:            true,
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
			// Treat 409 as success — adopt the existing account into state
			existing := r.findExistingAccount(ctx, cloudProvider, data)
			if existing == nil {
				resp.Diagnostics.AddError("Restore Account Already Exists",
					fmt.Sprintf("A restore account with this configuration already exists but could not be found via the API.\n\nOriginal error: %s", err.Error()))
				return
			}
			tflog.Info(ctx, "Restore account already exists (409 Conflict), adopting into state", map[string]interface{}{
				"id":   existing.Id,
				"name": existing.GetName(),
			})
			account = existing
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to connect restore account: %s", err))
			return
		}
	}

	// Update state from response
	data.Id = types.StringValue(account.Id)
	data.Status = types.StringValue(string(account.Status))
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

	data.CloudProvider = types.StringValue(string(account.RestoreAccountAttributes.GetCloudProvider()))

	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	// Populate role from API response (not plan data, which may be unknown during replace)
	data.Role = types.StringNull()
	if cloudProvider == CloudProviderAWS && account.RestoreAccountAttributes.HasAws() {
		awsAttrs := account.RestoreAccountAttributes.GetAws()
		data.Role = types.StringValue(awsAttrs.GetRoleArn())
		if data.Aws != nil {
			data.Aws.RoleArn = types.StringValue(awsAttrs.GetRoleArn())
		}
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

	account, err := r.client.GetRestoreAccount(ctx, data.Id.ValueString())
	if err != nil {
		// A deleted account returns 404 — drop it from state so Terraform plans a recreate.
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account: %s", err))
		return
	}

	data.Name = types.StringValue(account.GetName())
	data.Status = types.StringValue(string(account.Status))
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

	cloudProvider := CloudProvider(account.RestoreAccountAttributes.GetCloudProvider())
	data.CloudProvider = types.StringValue(cloudProvider.String())

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

	// Surface account statuses the provider cannot auto-remediate so the user
	// sees them in plan output. DISCONNECTED is handled by the plan modifier
	// and the Update flow; anything else non-CONNECTED needs manual attention.
	status := data.Status.ValueString()
	if status != "" && status != "CONNECTED" && status != "DISCONNECTED" {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("status"),
			"Restore Account Requires Manual Intervention",
			fmt.Sprintf(
				"Restore account %s is in status %q. The provider cannot automatically remediate this state; resolve the underlying issue in the Eon console or cloud provider and re-run.",
				data.Id.ValueString(), status,
			),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RestoreAccountResourceModel
	var state RestoreAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Azure restore accounts are immutable via the update API; any change to the
	// azure block must go through replacement. Reject it here with a clear error
	// rather than silently no-op'ing (the cloud_provider itself already forces
	// replacement, so this only fires for edits within the azure block).
	if azureAttributesChanged(plan.Azure, state.Azure) {
		resp.Diagnostics.AddError(
			"Azure Restore Account Update Not Supported",
			"Azure restore account attributes (tenant_id, subscription_id, resource_group_name) cannot be updated in place. "+
				"Recreate the resource (e.g. `terraform taint`) to apply these changes.",
		)
		return
	}

	accountId := state.Id.ValueString()
	var latestAccount *externalEonSdkAPI.RestoreAccount

	// Step 1: Update mutable fields (name, aws.role_arn, gcp.service_account) if changed.
	updateReq := r.buildUpdateRequest(plan, state)
	if updateReq != nil {
		tflog.Info(ctx, "Updating restore account", map[string]interface{}{
			"id": accountId,
		})

		account, err := r.client.UpdateRestoreAccount(ctx, accountId, *updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Update Failed",
				fmt.Sprintf("Unable to update restore account %s: %s", accountId, err),
			)
			return
		}
		latestAccount = account
	}

	// Step 2: Reconnect if the account is DISCONNECTED.
	if state.Status.ValueString() == "DISCONNECTED" {
		tflog.Info(ctx, "Restore account is disconnected, attempting reconnect", map[string]interface{}{
			"id": accountId,
		})

		account, err := r.client.ReconnectRestoreAccount(ctx, accountId)
		if err != nil {
			resp.Diagnostics.AddError(
				"Reconnect Failed",
				fmt.Sprintf("Unable to reconnect restore account %s: %s", accountId, err),
			)
			return
		}
		latestAccount = account

		tflog.Info(ctx, "Restore account reconnected", map[string]interface{}{
			"id": accountId,
		})
	}

	// Step 3: Build new state from the API response, or read back if no response was returned.
	if latestAccount == nil {
		fetched, err := r.readRestoreAccount(ctx, accountId, plan)
		if err != nil {
			resp.Diagnostics.AddError(
				"Read After Update Failed",
				fmt.Sprintf("Unable to read restore account %s after update: %s", accountId, err),
			)
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, fetched)...)
		return
	}

	newState := r.mapAccountToState(latestAccount, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// azureAttributesChanged reports whether any azure block attribute differs
// between plan and state. Restore accounts cannot update azure attributes in
// place, so such a change must force replacement.
func azureAttributesChanged(plan, state *AzureAccountConfigModel) bool {
	if plan == nil || state == nil {
		return false
	}
	return plan.TenantId.ValueString() != state.TenantId.ValueString() ||
		plan.SubscriptionId.ValueString() != state.SubscriptionId.ValueString() ||
		plan.ResourceGroupName.ValueString() != state.ResourceGroupName.ValueString()
}

// buildUpdateRequest compares plan and state and returns an UpdateRestoreAccountRequest
// if any mutable fields changed. Returns nil if nothing needs updating.
func (r *RestoreAccountResource) buildUpdateRequest(plan, state RestoreAccountResourceModel) *client.UpdateRestoreAccountRequest {
	var req client.UpdateRestoreAccountRequest
	var changed bool

	if plan.Name.ValueString() != state.Name.ValueString() {
		name := plan.Name.ValueString()
		req.Name = &name
		changed = true
	}

	if plan.Aws != nil && state.Aws != nil &&
		plan.Aws.RoleArn.ValueString() != state.Aws.RoleArn.ValueString() {
		roleArn := plan.Aws.RoleArn.ValueString()
		req.RestoreAccountAttributes = &client.UpdateRestoreAccountAttributes{
			Aws: &client.UpdateAwsRestoreAccountAttributes{
				RoleArn: &roleArn,
			},
		}
		changed = true
	}

	if plan.Gcp != nil && state.Gcp != nil &&
		plan.Gcp.ServiceAccount.ValueString() != state.Gcp.ServiceAccount.ValueString() {
		serviceAccount := plan.Gcp.ServiceAccount.ValueString()
		req.RestoreAccountAttributes = &client.UpdateRestoreAccountAttributes{
			Gcp: &client.UpdateGcpRestoreAccountAttributes{
				ServiceAccount: &serviceAccount,
			},
		}
		changed = true
	}

	if !changed {
		return nil
	}
	return &req
}

// mapAccountToState maps a RestoreAccount API response to the Terraform resource model.
// The plan is used to preserve fields not returned by the API (timestamps).
func (r *RestoreAccountResource) mapAccountToState(account *externalEonSdkAPI.RestoreAccount, plan RestoreAccountResourceModel) *RestoreAccountResourceModel {
	data := RestoreAccountResourceModel{
		Id:                types.StringValue(account.Id),
		Name:              types.StringValue(account.GetName()),
		Status:            types.StringValue(string(account.Status)),
		ProviderAccountId: types.StringValue(account.GetProviderAccountId()),
		CreatedAt:         plan.CreatedAt,
		UpdatedAt:         types.StringValue(time.Now().Format(time.RFC3339)),
	}

	cloudProvider := CloudProvider(account.RestoreAccountAttributes.GetCloudProvider())
	data.CloudProvider = types.StringValue(cloudProvider.String())

	switch cloudProvider {
	case CloudProviderAWS:
		if account.RestoreAccountAttributes.HasAws() {
			awsAttrs := account.RestoreAccountAttributes.GetAws()
			data.Aws = &AwsAccountConfigModel{
				RoleArn: types.StringValue(awsAttrs.GetRoleArn()),
			}
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
		data.Role = types.StringNull()
	case CloudProviderGCP:
		if account.RestoreAccountAttributes.HasGcp() {
			gcpAttrs := account.RestoreAccountAttributes.GetGcp()
			data.Gcp = &GcpAccountConfigModel{
				ProjectId:      types.StringValue(account.GetProviderAccountId()),
				ServiceAccount: types.StringValue(gcpAttrs.GetServiceAccount()),
			}
		}
		data.Role = types.StringNull()
	}

	return &data
}

// readRestoreAccount fetches a restore account by ID and maps it to the resource model.
func (r *RestoreAccountResource) readRestoreAccount(ctx context.Context, accountId string, plan RestoreAccountResourceModel) (*RestoreAccountResourceModel, error) {
	account, err := r.client.GetRestoreAccount(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("unable to get restore account: %w", err)
	}
	return r.mapAccountToState(account, plan), nil
}

func (r *RestoreAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RestoreAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.Id.ValueString()

	if strings.EqualFold(data.Status.ValueString(), "CONNECTED") {
		tflog.Debug(ctx, "Disconnecting restore account before delete", map[string]interface{}{"id": id})
		if err := r.client.DisconnectRestoreAccount(ctx, id); err != nil {
			tflog.Warn(ctx, "Disconnect failed during delete; proceeding to delete anyway", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			resp.Diagnostics.AddWarning(
				"Disconnect Failed",
				fmt.Sprintf("Could not disconnect restore account before delete (proceeding with delete): %s", err),
			)
		}
	}

	tflog.Debug(ctx, "Deleting restore account", map[string]interface{}{"id": id})
	if err := r.client.DeleteRestoreAccount(ctx, id); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete restore account: %s", err))
		return
	}

	tflog.Debug(ctx, "Restore account deleted", map[string]interface{}{"id": id})
}

func (r *RestoreAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	account, err := r.client.GetRestoreAccount(ctx, req.ID)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.Diagnostics.AddError(
				"Resource Not Found",
				fmt.Sprintf("Restore account with ID %s not found", req.ID),
			)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account during import: %s", err))
		return
	}

	var data RestoreAccountResourceModel
	data.Id = types.StringValue(account.Id)
	data.Name = types.StringValue(account.GetName())
	data.Status = types.StringValue(string(account.Status))
	data.ProviderAccountId = types.StringValue(account.GetProviderAccountId())

	cloudProvider := CloudProvider(account.RestoreAccountAttributes.GetCloudProvider())
	data.CloudProvider = types.StringValue(cloudProvider.String())

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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Successfully imported restore account", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
		"status": data.Status.ValueString(),
	})
}

// findExistingAccount attempts to find an existing restore account
// that matches the given configuration. Returns nil if not found.
func (r *RestoreAccountResource) findExistingAccount(ctx context.Context, cloudProvider CloudProvider, data RestoreAccountResourceModel) *externalEonSdkAPI.RestoreAccount {
	accounts, err := r.client.ListRestoreAccounts(ctx)
	if err != nil {
		tflog.Debug(ctx, "Failed to list restore accounts to find existing account", map[string]any{
			"error": err.Error(),
		})
		return nil
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
					return &account
				}
			}
		case CloudProviderAzure:
			if data.Azure != nil && account.RestoreAccountAttributes.HasAzure() {
				if account.GetProviderAccountId() == data.Azure.SubscriptionId.ValueString() {
					return &account
				}
			}
		case CloudProviderGCP:
			if data.Gcp != nil && account.RestoreAccountAttributes.HasGcp() {
				if account.GetProviderAccountId() == data.Gcp.ProjectId.ValueString() {
					return &account
				}
			}
		}
	}

	return nil
}
