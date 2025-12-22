package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &VaultResource{}
var _ resource.ResourceWithImportState = &VaultResource{}

func NewVaultResource() resource.Resource {
	return &VaultResource{}
}

type VaultResource struct {
	client *client.EonClient
}

type VaultResourceModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Region            types.String `tfsdk:"region"`
	CloudProvider     types.String `tfsdk:"cloud_provider"`
	AwsKmsKeyArn      types.String `tfsdk:"aws_kms_key_arn"`
	VaultAccountId    types.String `tfsdk:"vault_account_id"`
	ProviderAccountId types.String `tfsdk:"provider_account_id"`
	IsManagedByEon    types.Bool   `tfsdk:"is_managed_by_eon"`
}

// VaultUserInput contains only user-configurable fields for vault creation.
// When you add a field here, you MUST handle it in:
//   - ToUserInput() method (extraction from model)
//   - ToCreateRequest() method (building API request)
//   - MatchesVault() method (validation) - and add to test
type VaultUserInput struct {
	Name          string
	Region        string
	CloudProvider string
	AwsKmsKeyArn  *string
}

// ToUserInput extracts user-configurable fields from the Terraform model.
// If you add a field to VaultUserInput, the compiler will warn about missing field in struct literal.
func (m *VaultResourceModel) ToUserInput() VaultUserInput {
	input := VaultUserInput{
		Name:          m.Name.ValueString(),
		Region:        m.Region.ValueString(),
		CloudProvider: m.CloudProvider.ValueString(),
	}

	if !m.AwsKmsKeyArn.IsNull() && m.AwsKmsKeyArn.ValueString() != "" {
		kms := m.AwsKmsKeyArn.ValueString()
		input.AwsKmsKeyArn = &kms
	}

	return input
}

// ToCreateRequest builds a CreateVaultRequest from user input.
// If you add a field to VaultUserInput, the compiler may warn about unused field.
func (input *VaultUserInput) ToCreateRequest() (*externalEonSdkAPI.CreateVaultRequest, error) {
	cloudProvider := externalEonSdkAPI.Provider(input.CloudProvider)
	if cloudProvider != externalEonSdkAPI.AWS && cloudProvider != externalEonSdkAPI.AZURE && cloudProvider != externalEonSdkAPI.GCP {
		return nil, fmt.Errorf("cloud_provider must be one of: AWS, AZURE, GCP. Got: %s", input.CloudProvider)
	}

	vaultAttributes := externalEonSdkAPI.NewVaultProviderAttributesInput(cloudProvider)

	// Handle AWS-specific configuration
	if cloudProvider == externalEonSdkAPI.AWS {
		awsConfig := externalEonSdkAPI.NewAwsVaultConfigInput()

		if input.AwsKmsKeyArn != nil {
			awsConfig.SetEncryptionKey(*input.AwsKmsKeyArn)
		}

		vaultAttributes.SetAws(*awsConfig)
	}

	createReq := externalEonSdkAPI.NewCreateVaultRequest(
		input.Name,
		input.Region,
		*vaultAttributes,
	)

	return createReq, nil
}

// MatchesVault validates that this user input matches an existing vault.
// If you add a field to VaultUserInput, you MUST add validation here.
func (input *VaultUserInput) MatchesVault(vault *externalEonSdkAPI.BackupVault) (bool, string) {
	if vault.Name != input.Name {
		return false, fmt.Sprintf("name mismatch: existing vault is named '%s', but you requested '%s'",
			vault.Name, input.Name)
	}

	if string(vault.VaultAttributes.CloudProvider) != input.CloudProvider {
		return false, fmt.Sprintf("cloud provider mismatch: existing=%s, requested=%s",
			vault.VaultAttributes.CloudProvider, input.CloudProvider)
	}

	if vault.Region != input.Region {
		return false, fmt.Sprintf("region mismatch: existing=%s, requested=%s",
			vault.Region, input.Region)
	}

	if input.AwsKmsKeyArn != nil {
		var existingKms string
		if vault.VaultAttributes.Aws.IsSet() {
			if awsConfig := vault.VaultAttributes.Aws.Get(); awsConfig.EncryptionKey != nil {
				existingKms = *awsConfig.EncryptionKey
			}
		}

		if existingKms != *input.AwsKmsKeyArn {
			return false, fmt.Sprintf("AWS KMS key mismatch: existing=%s, requested=%s",
				existingKms, *input.AwsKmsKeyArn)
		}
	}

	return true, ""
}

func (r *VaultResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (r *VaultResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates and manages a backup vault for storing Eon backups. **Note**: Vaults are permanent and cannot be deleted. Running `terraform destroy` will only remove the vault from Terraform state; the actual vault will continue to exist in Eon permanently.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Vault identifier (UUID).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Vault display name in Eon.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region where the vault is hosted (e.g., `us-east-1`, `eu-central-1`).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cloud_provider": schema.StringAttribute{
				MarkdownDescription: "Cloud provider. Possible values: `AWS`, `AZURE`, `GCP`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"aws_kms_key_arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the AWS KMS customer-managed key (CMK) for encryption. If omitted, Eon uses its own managed encryption key. **Only applicable for AWS vaults.**",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"vault_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the vault account.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"provider_account_id": schema.StringAttribute{
				MarkdownDescription: "Cloud provider-assigned ID of the vault account.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_managed_by_eon": schema.BoolAttribute{
				MarkdownDescription: "Whether the vault is in an Eon-managed vault account.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *VaultResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VaultResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userInput := data.ToUserInput()

	// Build create request
	createReq, err := userInput.ToCreateRequest()
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	tflog.Debug(ctx, "Creating vault", map[string]interface{}{
		"name":           userInput.Name,
		"region":         userInput.Region,
		"cloud_provider": userInput.CloudProvider,
		"has_cmk":        userInput.AwsKmsKeyArn != nil,
	})

	vault, err := r.client.CreateVault(ctx, *createReq)
	if err != nil {
		var apiErr *client.APIError
		isAlreadyExists := errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict

		// If not a "already exists" error, fail immediately
		if !isAlreadyExists {
			resp.Diagnostics.AddError(
				"Failed to Create Vault",
				fmt.Sprintf("Unable to create vault: %s", err))
			return
		}

		// Vault already exists - try to auto-import
		tflog.Warn(ctx, "Vault creation returned 'already exists', attempting to find and import", map[string]interface{}{
			"name":   userInput.Name,
			"region": userInput.Region,
		})

		existingVault, findErr := r.findEonManagedVault(ctx,
			userInput.Region,
			userInput.CloudProvider)

		if findErr != nil {
			resp.Diagnostics.AddError(
				"Vault Already Exists",
				fmt.Sprintf("An Eon-managed vault for cloud provider '%s' in region '%s' already exists, but could not be retrieved for automatic import.\n\n"+
					"To resolve this, you can:\n"+
					"1. Manually import the existing vault using: terraform import eon_vault.<resource_name> <vault_id>\n"+
					"2. Choose a different region or cloud provider in your configuration\n\n"+
					"Note: Only one Eon-managed vault is allowed per (cloud provider + region + cloud account) combination.\n\n"+
					"Original error: %s\n"+
					"Lookup error: %s",
					userInput.CloudProvider,
					userInput.Region,
					err.Error(),
					findErr.Error()),
			)
			return
		}

		// Validate that the existing vault matches the requested configuration
		matches, mismatchReason := userInput.MatchesVault(existingVault)
		if !matches {
			resp.Diagnostics.AddError(
				"Vault Configuration Mismatch",
				fmt.Sprintf("An Eon-managed vault for cloud provider '%s' in region '%s' already exists (ID: %s, Name: '%s'), "+
					"but its configuration doesn't match your request.\n\n"+
					"Mismatch: %s\n\n"+
					"Existing vault details:\n"+
					"  Name: %s\n"+
					"  Cloud Account: %s\n\n"+
					"To resolve this, you can:\n"+
					"1. Update your Terraform configuration to match the existing vault (name: '%s')\n"+
					"2. Import the existing vault: terraform import eon_vault.<resource_name> %s\n"+
					"3. Choose a different region or cloud provider\n\n"+
					"Note: Only one Eon-managed vault is allowed per (cloud provider + region + cloud account) combination.",
					existingVault.VaultAttributes.CloudProvider,
					existingVault.Region,
					existingVault.Id,
					existingVault.Name,
					mismatchReason,
					existingVault.Name,
					existingVault.ProviderAccountId,
					existingVault.Name,
					existingVault.Id),
			)
			return
		}

		// Configuration matches - automatically import the vault
		resp.Diagnostics.AddWarning(
			"Vault Already Exists - Automatically Imported",
			fmt.Sprintf("An Eon-managed vault for cloud provider '%s' in region '%s' already exists.\n\n"+
				"Vault details:\n"+
				"  ID: %s\n"+
				"  Name: %s\n"+
				"  Cloud Account: %s\n\n"+
				"The existing vault matches your configuration and has been automatically imported into Terraform state. "+
				"This is expected behavior for permanent resources like vaults.\n\n"+
				"Note: Only one Eon-managed vault is allowed per (cloud provider + region + cloud account) combination.",
				existingVault.VaultAttributes.CloudProvider,
				existingVault.Region,
				existingVault.Id,
				existingVault.Name,
				existingVault.ProviderAccountId),
		)

		tflog.Info(ctx, "Successfully imported existing vault", map[string]interface{}{
			"id":     existingVault.Id,
			"name":   existingVault.Name,
			"region": existingVault.Region,
		})

		vault = existingVault
	} else {
		tflog.Info(ctx, "Successfully created new vault", map[string]interface{}{
			"id":     vault.Id,
			"name":   vault.Name,
			"region": vault.Region,
		})
	}

	// Populate state from response (works for both create and import)
	data.Id = types.StringValue(vault.Id)
	data.VaultAccountId = types.StringValue(vault.VaultAccountId)
	data.ProviderAccountId = types.StringValue(vault.ProviderAccountId)
	data.IsManagedByEon = types.BoolValue(vault.IsManagedByEon)

	// Extract encryption key from response if present
	if vault.VaultAttributes.Aws.IsSet() {
		awsConfig := vault.VaultAttributes.Aws.Get()
		if awsConfig.EncryptionKey != nil {
			data.AwsKmsKeyArn = types.StringValue(*awsConfig.EncryptionKey)
		} else {
			data.AwsKmsKeyArn = types.StringNull()
		}
	}

	tflog.Debug(ctx, "Vault state updated", map[string]interface{}{
		"id":                  data.Id.ValueString(),
		"vault_account_id":    data.VaultAccountId.ValueString(),
		"provider_account_id": data.ProviderAccountId.ValueString(),
		"is_managed_by_eon":   data.IsManagedByEon.ValueBool(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VaultResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vault, err := r.client.GetVault(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault: %s", err))
		return
	}

	// Update state from API response
	data.Name = types.StringValue(vault.Name)
	data.Region = types.StringValue(vault.Region)
	data.VaultAccountId = types.StringValue(vault.VaultAccountId)
	data.ProviderAccountId = types.StringValue(vault.ProviderAccountId)
	data.IsManagedByEon = types.BoolValue(vault.IsManagedByEon)
	data.CloudProvider = types.StringValue(string(vault.VaultAttributes.CloudProvider))

	// Extract encryption key from response if present
	if vault.VaultAttributes.Aws.IsSet() {
		awsConfig := vault.VaultAttributes.Aws.Get()
		if awsConfig.EncryptionKey != nil {
			data.AwsKmsKeyArn = types.StringValue(*awsConfig.EncryptionKey)
		} else {
			data.AwsKmsKeyArn = types.StringNull()
		}
	} else {
		data.AwsKmsKeyArn = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VaultResourceModel
	var state VaultResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only name can be updated - other attributes are immutable and marked with RequiresReplace
	if !plan.Name.Equal(state.Name) {
		updateReq := externalEonSdkAPI.NewUpdateVaultRequest(plan.Name.ValueString())

		tflog.Debug(ctx, "Updating vault name", map[string]interface{}{
			"id":       state.Id.ValueString(),
			"old_name": state.Name.ValueString(),
			"new_name": plan.Name.ValueString(),
		})

		vault, err := r.client.UpdateVault(ctx, state.Id.ValueString(), *updateReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update vault: %s", err))
			return
		}

		// Update state with the new name from response
		plan.Name = types.StringValue(vault.Name)

		tflog.Debug(ctx, "Vault name updated", map[string]interface{}{
			"id":   state.Id.ValueString(),
			"name": vault.Name,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VaultResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Vaults are permanent and cannot be deleted - remove from state only
	resp.Diagnostics.AddWarning(
		"Vault Not Deleted",
		fmt.Sprintf("The vault '%s' (ID: %s) has been removed from Terraform state but still exists in Eon. Vaults are permanent and cannot be deleted via API, Terraform, or console.",
			data.Name.ValueString(),
			data.Id.ValueString()),
	)

	tflog.Warn(ctx, "Removing vault from Terraform state only - vaults are permanent", map[string]interface{}{
		"id":   data.Id.ValueString(),
		"name": data.Name.ValueString(),
		"note": "The actual vault will continue to exist in Eon permanently. Vaults cannot be deleted.",
	})

	// The vault is automatically removed from state when this function completes successfully
	// No API call is made - deletion is not supported
}

func (r *VaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Set the ID from the import request
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	// Fetch the vault details
	vault, err := r.client.GetVault(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vault during import: %s", err))
		return
	}

	// Populate state
	var data VaultResourceModel
	data.Id = types.StringValue(vault.Id)
	data.Name = types.StringValue(vault.Name)
	data.Region = types.StringValue(vault.Region)
	data.VaultAccountId = types.StringValue(vault.VaultAccountId)
	data.ProviderAccountId = types.StringValue(vault.ProviderAccountId)
	data.IsManagedByEon = types.BoolValue(vault.IsManagedByEon)
	data.CloudProvider = types.StringValue(string(vault.VaultAttributes.CloudProvider))

	// Extract encryption key if present
	if vault.VaultAttributes.Aws.IsSet() {
		awsConfig := vault.VaultAttributes.Aws.Get()
		if awsConfig.EncryptionKey != nil {
			data.AwsKmsKeyArn = types.StringValue(*awsConfig.EncryptionKey)
		} else {
			data.AwsKmsKeyArn = types.StringNull()
		}
	} else {
		data.AwsKmsKeyArn = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Successfully imported vault", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
		"region": data.Region.ValueString(),
	})
}

// findEonManagedVault searches for an Eon-managed vault with the specified region and cloud provider
// Note: Only one Eon-managed vault can exist per (region + cloud provider + provider account ID) combination
func (r *VaultResource) findEonManagedVault(
	ctx context.Context,
	region string,
	cloudProvider string,
) (*externalEonSdkAPI.BackupVault, error) {
	tflog.Debug(ctx, "Searching for Eon-managed vault", map[string]interface{}{
		"region":         region,
		"cloud_provider": cloudProvider,
	})

	vaults, err := r.client.ListVaults(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list vaults: %w", err)
	}

	for _, vault := range vaults {
		if vault.Region == region &&
			string(vault.VaultAttributes.CloudProvider) == cloudProvider &&
			vault.IsManagedByEon {
			tflog.Debug(ctx, "Found existing Eon-managed vault", map[string]interface{}{
				"id":                  vault.Id,
				"name":                vault.Name,
				"region":              vault.Region,
				"cloud_provider":      vault.VaultAttributes.CloudProvider,
				"provider_account_id": vault.ProviderAccountId,
			})
			return &vault, nil
		}
	}

	return nil, fmt.Errorf("no Eon-managed vault found for region=%s cloud_provider=%s", region, cloudProvider)
}
