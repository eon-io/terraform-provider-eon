package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	gcsBucketConfigurationMethodAuto          = "AUTO"
	gcsBucketConfigurationMethodForceEnabled  = "FORCE_ENABLED"
	gcsBucketConfigurationMethodForceDisabled = "FORCE_DISABLED"
)

var _ resource.Resource = &GcsBucketConfigurationResource{}
var _ resource.ResourceWithImportState = &GcsBucketConfigurationResource{}

func NewGcsBucketConfigurationResource() resource.Resource {
	return &GcsBucketConfigurationResource{}
}

type GcsBucketConfigurationResource struct {
	client *client.EonClient
}

type GcsBucketConfigurationResourceModel struct {
	Id                  types.String `tfsdk:"id"`
	ResourceId          types.String `tfsdk:"resource_id"`
	ScanDetectionMethod types.String `tfsdk:"scan_detection_method"`
	FullScanMethod      types.String `tfsdk:"full_scan_method"`
}

func (r *GcsBucketConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gcs_bucket_configuration"
}

func (r *GcsBucketConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages GCS bucket backup method configuration for an inventory resource. " +
			"`scan_detection_method` controls event-notification based scan detection. " +
			"`full_scan_method` controls inventory-based full scans. Deleting this resource resets both methods to `AUTO`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource. Mirrors `resource_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the GCS bucket inventory resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"scan_detection_method": schema.StringAttribute{
				MarkdownDescription: "Scan detection method. Supported values: `AUTO`, `FORCE_ENABLED`, `FORCE_DISABLED`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(gcsBucketConfigurationMethodAuto),
			},
			"full_scan_method": schema.StringAttribute{
				MarkdownDescription: "Full-scan method. Supported values: `AUTO`, `FORCE_ENABLED`, `FORCE_DISABLED`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(gcsBucketConfigurationMethodAuto),
			},
		},
	}
}

func (r *GcsBucketConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GcsBucketConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GcsBucketConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GcsBucketConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GcsBucketConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetCloudResourceConfiguration(ctx, data.ResourceId.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read GCS bucket configuration: %s", err))
		return
	}

	if diags := cloudResourceConfigurationToModel(config, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GcsBucketConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GcsBucketConfigurationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GcsBucketConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GcsBucketConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	autoConfig, diags := backupMethodToAPIConfig(gcsBucketConfigurationMethodAuto)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateCloudResourceConfiguration(ctx, data.ResourceId.ValueString(), client.UpdateCloudResourceConfigurationRequest{
		CdcBackup:       &autoConfig,
		InventoryBackup: &autoConfig,
	})
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset GCS bucket configuration: %s", err))
		return
	}
}

func (r *GcsBucketConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), req.ID)...)
}

func (r *GcsBucketConfigurationResource) apply(ctx context.Context, data *GcsBucketConfigurationResourceModel, diags *diag.Diagnostics) {
	updateReq, d := gcsBucketConfigurationModelToUpdateRequest(data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating GCS bucket configuration", map[string]interface{}{
		"resource_id": data.ResourceId.ValueString(),
	})

	if err := r.client.UpdateCloudResourceConfiguration(ctx, data.ResourceId.ValueString(), updateReq); err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to update GCS bucket configuration: %s", err))
		return
	}

	data.Id = data.ResourceId
	data.ScanDetectionMethod = types.StringValue(normalizeBackupMethod(data.ScanDetectionMethod.ValueString()))
	data.FullScanMethod = types.StringValue(normalizeBackupMethod(data.FullScanMethod.ValueString()))
}

func gcsBucketConfigurationModelToUpdateRequest(data *GcsBucketConfigurationResourceModel) (client.UpdateCloudResourceConfigurationRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := client.UpdateCloudResourceConfigurationRequest{}

	cdcBackup, d := backupMethodToAPIConfig(data.ScanDetectionMethod.ValueString())
	diags.Append(d...)
	req.CdcBackup = &cdcBackup

	inventoryBackup, d := backupMethodToAPIConfig(data.FullScanMethod.ValueString())
	diags.Append(d...)
	req.InventoryBackup = &inventoryBackup

	return req, diags
}

func cloudResourceConfigurationToModel(config *client.CloudResourceConfiguration, data *GcsBucketConfigurationResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	scanDetectionMethod, d := apiConfigToBackupMethod("scan_detection_method", config.CdcBackup)
	diags.Append(d...)
	fullScanMethod, d := apiConfigToBackupMethod("full_scan_method", config.InventoryBackup)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	data.Id = data.ResourceId
	data.ScanDetectionMethod = types.StringValue(scanDetectionMethod)
	data.FullScanMethod = types.StringValue(fullScanMethod)
	return diags
}

func backupMethodToAPIConfig(method string) (client.CloudResourceBackupMethodConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	normalized := normalizeBackupMethod(method)
	switch normalized {
	case gcsBucketConfigurationMethodAuto:
		systemControlled := true
		return client.CloudResourceBackupMethodConfig{
			SystemControlled: &systemControlled,
		}, diags
	case gcsBucketConfigurationMethodForceEnabled:
		enabled := true
		systemControlled := false
		return client.CloudResourceBackupMethodConfig{
			Enabled:          &enabled,
			SystemControlled: &systemControlled,
		}, diags
	case gcsBucketConfigurationMethodForceDisabled:
		enabled := false
		systemControlled := false
		return client.CloudResourceBackupMethodConfig{
			Enabled:          &enabled,
			SystemControlled: &systemControlled,
		}, diags
	default:
		diags.AddError(
			"Invalid GCS Bucket Configuration Method",
			fmt.Sprintf("Expected one of %s, %s, %s, got: %s",
				gcsBucketConfigurationMethodAuto,
				gcsBucketConfigurationMethodForceEnabled,
				gcsBucketConfigurationMethodForceDisabled,
				method,
			),
		)
		return client.CloudResourceBackupMethodConfig{}, diags
	}
}

func apiConfigToBackupMethod(attributeName string, config client.CloudResourceBackupMethodConfig) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if config.SystemControlled == nil {
		diags.AddError(
			"Invalid GCS Bucket Configuration Response",
			fmt.Sprintf("API response for %s did not include systemControlled", attributeName),
		)
		return "", diags
	}

	if *config.SystemControlled {
		return gcsBucketConfigurationMethodAuto, diags
	}

	if config.Enabled == nil {
		diags.AddError(
			"Invalid GCS Bucket Configuration Response",
			fmt.Sprintf("API response for %s is user-controlled but did not include enabled", attributeName),
		)
		return "", diags
	}

	if *config.Enabled {
		return gcsBucketConfigurationMethodForceEnabled, diags
	}

	return gcsBucketConfigurationMethodForceDisabled, diags
}

func normalizeBackupMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}
