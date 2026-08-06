package provider

import (
	"context"
	"errors"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &RestoreAccountMetricsConfigResource{}
var _ resource.ResourceWithImportState = &RestoreAccountMetricsConfigResource{}

func NewRestoreAccountMetricsConfigResource() resource.Resource {
	return &RestoreAccountMetricsConfigResource{}
}

type RestoreAccountMetricsConfigResource struct {
	client *client.EonClient
}

// RestoreAccountMetricsConfigResourceModel is the Terraform model for a restore account's
// metrics configuration. The configuration is a singleton per restore account, so the
// resource id mirrors restore_account_id.
type RestoreAccountMetricsConfigResourceModel struct {
	Id               types.String                `tfsdk:"id"`
	RestoreAccountId types.String                `tfsdk:"restore_account_id"`
	Aws              *awsAccountMetricsDestModel `tfsdk:"aws"`
}

type awsAccountMetricsDestModel struct {
	Region types.String `tfsdk:"region"`
}

func (r *RestoreAccountMetricsConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_metrics_config"
}

func (r *RestoreAccountMetricsConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the job metrics configuration of a restore account. When enabled, Eon publishes restore-job metrics to the configured destination (for example CloudWatch). The configuration is a singleton per restore account; deleting this resource disables metrics for the account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account. Mirrors `restore_account_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account whose metrics configuration is managed.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
		Blocks: map[string]schema.Block{
			"aws": schema.SingleNestedBlock{
				MarkdownDescription: "AWS metrics destination. Set when publishing metrics to CloudWatch in the restore account.",
				Attributes: map[string]schema.Attribute{
					"region": schema.StringAttribute{
						MarkdownDescription: "CloudWatch region to publish metrics to.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *RestoreAccountMetricsConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.EonClient, got: %T", req.ProviderData))
		return
	}

	r.client = c
}

func (r *RestoreAccountMetricsConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RestoreAccountMetricsConfigResourceModel

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

func (r *RestoreAccountMetricsConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RestoreAccountMetricsConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetRestoreAccountMetricsConfig(ctx, data.RestoreAccountId.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account metrics config: %s", err))
		return
	}

	if !config.GetEnabled() {
		tflog.Debug(ctx, "Restore account metrics are disabled; removing from state", map[string]interface{}{
			"restore_account_id": data.RestoreAccountId.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(config.GetRestoreAccountId())
	data.RestoreAccountId = types.StringValue(config.GetRestoreAccountId())
	resp.Diagnostics.Append(metricsConfigToModel(config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreAccountMetricsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RestoreAccountMetricsConfigResourceModel

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

func (r *RestoreAccountMetricsConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RestoreAccountMetricsConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Disabling restore account metrics config", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
	})

	err := r.client.DisableRestoreAccountMetricsConfig(ctx, data.RestoreAccountId.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disable restore account metrics config: %s", err))
		return
	}
}

func (r *RestoreAccountMetricsConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("restore_account_id"), req.ID)...)
}

func (r *RestoreAccountMetricsConfigResource) apply(ctx context.Context, data *RestoreAccountMetricsConfigResourceModel, diags *diag.Diagnostics) {
	enableReq := modelToEnableMetricsRequest(data)

	tflog.Debug(ctx, "Enabling restore account metrics config", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
	})

	config, err := r.client.EnableRestoreAccountMetricsConfig(ctx, data.RestoreAccountId.ValueString(), enableReq)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to enable restore account metrics config: %s", err))
		return
	}

	data.Id = types.StringValue(config.GetRestoreAccountId())
	data.RestoreAccountId = types.StringValue(config.GetRestoreAccountId())
	diags.Append(metricsConfigToModel(config, data)...)
}

func modelToEnableMetricsRequest(data *RestoreAccountMetricsConfigResourceModel) externalEonSdkAPI.EnableRestoreAccountMetricsConfigRequest {
	req := externalEonSdkAPI.EnableRestoreAccountMetricsConfigRequest{}
	if data.Aws != nil {
		awsDest := externalEonSdkAPI.NewAwsAccountMetricsDestination()
		if !data.Aws.Region.IsNull() && !data.Aws.Region.IsUnknown() && data.Aws.Region.ValueString() != "" {
			awsDest.SetRegion(data.Aws.Region.ValueString())
		}
		req.SetAws(*awsDest)
	}
	return req
}

func metricsConfigToModel(config *externalEonSdkAPI.RestoreAccountMetricsConfig, data *RestoreAccountMetricsConfigResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Aws = nil
	destination := config.GetDestination()
	if destination.HasAws() {
		aws := destination.GetAws()
		model := &awsAccountMetricsDestModel{
			Region: types.StringNull(),
		}
		if aws.HasRegion() {
			model.Region = types.StringValue(aws.GetRegion())
		}
		data.Aws = model
	}

	return diags
}
