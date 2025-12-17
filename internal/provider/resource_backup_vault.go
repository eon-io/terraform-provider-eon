package provider

import (
	"context"
	"fmt"

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

var _ resource.Resource = &BackupVaultResource{}
var _ resource.ResourceWithImportState = &BackupVaultResource{}

func NewBackupVaultResource() resource.Resource {
	return &BackupVaultResource{}
}

type BackupVaultResource struct {
	client *client.EonClient
}

type BackupVaultResourceModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Region            types.String `tfsdk:"region"`
	CloudProvider     types.String `tfsdk:"cloud_provider"`
	AwsKmsKeyArn      types.String `tfsdk:"aws_kms_key_arn"`
	VaultAccountId    types.String `tfsdk:"vault_account_id"`
	ProviderAccountId types.String `tfsdk:"provider_account_id"`
	IsManagedByEon    types.Bool   `tfsdk:"is_managed_by_eon"`
}

func (r *BackupVaultResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_vault"
}

func (r *BackupVaultResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				MarkdownDescription: "Cloud provider. Possible values: `AWS`, `AZURE`, `GCP`. Currently only `AWS` is fully supported.",
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

func (r *BackupVaultResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BackupVaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BackupVaultResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate cloud provider
	cloudProvider := externalEonSdkAPI.Provider(data.CloudProvider.ValueString())
	if cloudProvider != externalEonSdkAPI.AWS && cloudProvider != externalEonSdkAPI.AZURE && cloudProvider != externalEonSdkAPI.GCP {
		resp.Diagnostics.AddError(
			"Invalid Cloud Provider",
			fmt.Sprintf("cloud_provider must be one of: AWS, AZURE, GCP. Got: %s", data.CloudProvider.ValueString()),
		)
		return
	}

	// Build vault attributes based on cloud provider
	vaultAttributes := externalEonSdkAPI.NewVaultProviderAttributesInput(cloudProvider)

	// Handle AWS-specific configuration
	if cloudProvider == externalEonSdkAPI.AWS {
		awsConfig := externalEonSdkAPI.NewAwsVaultConfigInput()

		// Only set encryption key if provided
		if !data.AwsKmsKeyArn.IsNull() && data.AwsKmsKeyArn.ValueString() != "" {
			awsConfig.SetEncryptionKey(data.AwsKmsKeyArn.ValueString())
		}

		vaultAttributes.SetAws(*awsConfig)
	}

	// Build create request
	createReq := externalEonSdkAPI.NewCreateVaultRequest(
		data.Name.ValueString(),
		data.Region.ValueString(),
		*vaultAttributes,
	)

	tflog.Debug(ctx, "Creating backup vault", map[string]interface{}{
		"name":           data.Name.ValueString(),
		"region":         data.Region.ValueString(),
		"cloud_provider": data.CloudProvider.ValueString(),
		"has_cmk":        !data.AwsKmsKeyArn.IsNull(),
	})

	// Create the vault
	vault, err := r.client.CreateVault(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create backup vault: %s", err))
		return
	}

	// Populate state from response
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

	tflog.Debug(ctx, "Backup vault created", map[string]interface{}{
		"id":                  data.Id.ValueString(),
		"vault_account_id":    data.VaultAccountId.ValueString(),
		"provider_account_id": data.ProviderAccountId.ValueString(),
		"is_managed_by_eon":   data.IsManagedByEon.ValueBool(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupVaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BackupVaultResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vault, err := r.client.GetVault(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup vault: %s", err))
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

func (r *BackupVaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BackupVaultResourceModel
	var state BackupVaultResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only name can be updated - other attributes are immutable and marked with RequiresReplace
	if !plan.Name.Equal(state.Name) {
		updateReq := externalEonSdkAPI.NewUpdateVaultRequest(plan.Name.ValueString())

		tflog.Debug(ctx, "Updating backup vault name", map[string]interface{}{
			"id":       state.Id.ValueString(),
			"old_name": state.Name.ValueString(),
			"new_name": plan.Name.ValueString(),
		})

		vault, err := r.client.UpdateVault(ctx, state.Id.ValueString(), *updateReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update backup vault: %s", err))
			return
		}

		// Update state with the new name from response
		plan.Name = types.StringValue(vault.Name)

		tflog.Debug(ctx, "Backup vault name updated", map[string]interface{}{
			"id":   state.Id.ValueString(),
			"name": vault.Name,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BackupVaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BackupVaultResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Vaults are permanent and cannot be deleted - remove from state only
	resp.Diagnostics.AddWarning(
		"Vault Not Deleted",
		fmt.Sprintf("The backup vault '%s' (ID: %s) has been removed from Terraform state but still exists in Eon. Vaults are permanent and cannot be deleted via API, Terraform, or console.",
			data.Name.ValueString(),
			data.Id.ValueString()),
	)

	tflog.Warn(ctx, "Removing backup vault from Terraform state only - vaults are permanent", map[string]interface{}{
		"id":   data.Id.ValueString(),
		"name": data.Name.ValueString(),
		"note": "The actual vault will continue to exist in Eon permanently. Vaults cannot be deleted.",
	})

	// The vault is automatically removed from state when this function completes successfully
	// No API call is made - deletion is not supported
}

func (r *BackupVaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Set the ID from the import request
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	// Fetch the vault details
	vault, err := r.client.GetVault(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup vault during import: %s", err))
		return
	}

	// Populate state
	var data BackupVaultResourceModel
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

	tflog.Info(ctx, "Successfully imported backup vault", map[string]interface{}{
		"id":     data.Id.ValueString(),
		"name":   data.Name.ValueString(),
		"region": data.Region.ValueString(),
	})
}
