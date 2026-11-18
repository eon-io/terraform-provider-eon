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
	s3BucketModeAuto = "AUTO"

	s3ScanModeInventory   = "INVENTORY"
	s3ScanModeListObjects = "LIST_OBJECTS"

	s3ChangeDetectionModeScanAndChangeEvents = "SCAN_AND_CHANGE_EVENTS"
	s3ChangeDetectionModeScanOnly            = "SCAN_ONLY"
)

// s3BucketModeOption describes one Terraform attribute that maps onto an auto/user-controlled
// API setting: AUTO lets Eon decide, the other two values force the setting on or off.
type s3BucketModeOption struct {
	attributeName string
	enabledValue  string
	disabledValue string
}

var (
	s3ScanModeOption = s3BucketModeOption{
		attributeName: "scan_mode",
		enabledValue:  s3ScanModeInventory,
		disabledValue: s3ScanModeListObjects,
	}
	s3ChangeDetectionModeOption = s3BucketModeOption{
		attributeName: "change_detection_mode",
		enabledValue:  s3ChangeDetectionModeScanAndChangeEvents,
		disabledValue: s3ChangeDetectionModeScanOnly,
	}
)

var _ resource.Resource = &S3BucketConfigurationResource{}
var _ resource.ResourceWithImportState = &S3BucketConfigurationResource{}

func NewS3BucketConfigurationResource() resource.Resource {
	return &S3BucketConfigurationResource{}
}

type S3BucketConfigurationResource struct {
	client *client.EonClient
}

type S3BucketConfigurationResourceModel struct {
	Id                  types.String `tfsdk:"id"`
	ResourceId          types.String `tfsdk:"resource_id"`
	ScanMode            types.String `tfsdk:"scan_mode"`
	ChangeDetectionMode types.String `tfsdk:"change_detection_mode"`
}

func (r *S3BucketConfigurationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_bucket_configuration"
}

func (r *S3BucketConfigurationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages how Eon scans an Amazon S3 bucket and detects changes between scans. " +
			"`scan_mode` chooses how Eon lists the bucket's objects during a full scan. " +
			"`change_detection_mode` chooses whether Eon also uses S3 change events between scans. " +
			"Deleting this resource resets both settings to `AUTO`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource. Mirrors `resource_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the S3 bucket inventory resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"scan_mode": schema.StringAttribute{
				MarkdownDescription: "How Eon lists the bucket's objects during a full scan. Supported values: " +
					"`AUTO` (Eon decides), `INVENTORY` (read S3 Inventory reports), `LIST_OBJECTS` (call the S3 ListObjects API). " +
					"Defaults to `AUTO`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(s3BucketModeAuto),
			},
			"change_detection_mode": schema.StringAttribute{
				MarkdownDescription: "How Eon detects object changes between full scans. Supported values: " +
					"`AUTO` (Eon decides), `SCAN_AND_CHANGE_EVENTS` (scan plus S3 event notifications), `SCAN_ONLY` (full scans only). " +
					"Defaults to `AUTO`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(s3BucketModeAuto),
			},
		},
	}
}

func (r *S3BucketConfigurationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *S3BucketConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data S3BucketConfigurationResourceModel

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

func (r *S3BucketConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data S3BucketConfigurationResourceModel

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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read S3 bucket configuration: %s", err))
		return
	}

	if diags := s3BucketConfigurationToModel(config, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *S3BucketConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data S3BucketConfigurationResourceModel

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

func (r *S3BucketConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data S3BucketConfigurationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := s3BucketConfigurationModelToUpdateRequest(&S3BucketConfigurationResourceModel{
		ScanMode:            types.StringValue(s3BucketModeAuto),
		ChangeDetectionMode: types.StringValue(s3BucketModeAuto),
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateCloudResourceConfiguration(ctx, data.ResourceId.ValueString(), updateReq)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to reset S3 bucket configuration: %s", err))
		return
	}
}

func (r *S3BucketConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), req.ID)...)
}

func (r *S3BucketConfigurationResource) apply(ctx context.Context, data *S3BucketConfigurationResourceModel, diags *diag.Diagnostics) {
	updateReq, d := s3BucketConfigurationModelToUpdateRequest(data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating S3 bucket configuration", map[string]interface{}{
		"resource_id": data.ResourceId.ValueString(),
	})

	if err := r.client.UpdateCloudResourceConfiguration(ctx, data.ResourceId.ValueString(), updateReq); err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to update S3 bucket configuration: %s", err))
		return
	}

	data.Id = data.ResourceId
	data.ScanMode = types.StringValue(normalizeBackupMethod(data.ScanMode.ValueString()))
	data.ChangeDetectionMode = types.StringValue(normalizeBackupMethod(data.ChangeDetectionMode.ValueString()))
}

func s3BucketConfigurationModelToUpdateRequest(data *S3BucketConfigurationResourceModel) (client.UpdateCloudResourceConfigurationRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	inventoryBackup, d := s3ScanModeOption.toAPIConfig(data.ScanMode.ValueString())
	diags.Append(d...)
	cdcBackup, d := s3ChangeDetectionModeOption.toAPIConfig(data.ChangeDetectionMode.ValueString())
	diags.Append(d...)

	return client.UpdateCloudResourceConfigurationRequest{
		InventoryBackup: &inventoryBackup,
		CdcBackup:       &cdcBackup,
	}, diags
}

func s3BucketConfigurationToModel(config *client.CloudResourceConfiguration, data *S3BucketConfigurationResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	scanMode, d := s3ScanModeOption.fromAPIConfig(config.InventoryBackup)
	diags.Append(d...)
	changeDetectionMode, d := s3ChangeDetectionModeOption.fromAPIConfig(config.CdcBackup)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	data.Id = data.ResourceId
	data.ScanMode = types.StringValue(scanMode)
	data.ChangeDetectionMode = types.StringValue(changeDetectionMode)
	return diags
}

func (o s3BucketModeOption) toAPIConfig(value string) (client.CloudResourceBackupMethodConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	systemControlled := false
	switch normalizeBackupMethod(value) {
	case s3BucketModeAuto:
		systemControlled = true
		return client.CloudResourceBackupMethodConfig{SystemControlled: &systemControlled}, diags
	case o.enabledValue:
		enabled := true
		return client.CloudResourceBackupMethodConfig{Enabled: &enabled, SystemControlled: &systemControlled}, diags
	case o.disabledValue:
		enabled := false
		return client.CloudResourceBackupMethodConfig{Enabled: &enabled, SystemControlled: &systemControlled}, diags
	default:
		diags.AddAttributeError(
			path.Root(o.attributeName),
			"Invalid S3 Bucket Configuration Value",
			fmt.Sprintf("Expected one of %s, got: %s", strings.Join(o.allowedValues(), ", "), value),
		)
		return client.CloudResourceBackupMethodConfig{}, diags
	}
}

func (o s3BucketModeOption) fromAPIConfig(config client.CloudResourceBackupMethodConfig) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if config.SystemControlled == nil {
		diags.AddError(
			"Invalid S3 Bucket Configuration Response",
			fmt.Sprintf("API response for %s did not include systemControlled", o.attributeName),
		)
		return "", diags
	}
	if *config.SystemControlled {
		return s3BucketModeAuto, diags
	}

	if config.Enabled == nil {
		diags.AddError(
			"Invalid S3 Bucket Configuration Response",
			fmt.Sprintf("API response for %s is user-controlled but did not include enabled", o.attributeName),
		)
		return "", diags
	}
	if *config.Enabled {
		return o.enabledValue, diags
	}
	return o.disabledValue, diags
}

func (o s3BucketModeOption) allowedValues() []string {
	return []string{s3BucketModeAuto, o.enabledValue, o.disabledValue}
}
