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

var _ resource.Resource = &SourceAccountMetricsConfigResource{}
var _ resource.ResourceWithImportState = &SourceAccountMetricsConfigResource{}

func NewSourceAccountMetricsConfigResource() resource.Resource {
	return &SourceAccountMetricsConfigResource{}
}

type SourceAccountMetricsConfigResource struct {
	client *client.EonClient
}

// SourceAccountMetricsConfigResourceModel is the Terraform model for a source account's
// metrics configuration. The configuration is a singleton per source account, so the
// resource id mirrors source_account_id.
type SourceAccountMetricsConfigResourceModel struct {
	Id              types.String                `tfsdk:"id"`
	SourceAccountId types.String                `tfsdk:"source_account_id"`
	Aws             *awsAccountMetricsDestModel `tfsdk:"aws"`
}

func (r *SourceAccountMetricsConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_account_metrics_config"
}

func (r *SourceAccountMetricsConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the job metrics configuration of a source account. When enabled, Eon publishes backup-job metrics to the configured destination (for example CloudWatch). The configuration is a singleton per source account; deleting this resource disables metrics for the account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the source account. Mirrors `source_account_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the source account whose metrics configuration is managed.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
		Blocks: map[string]schema.Block{
			"aws": schema.SingleNestedBlock{
				MarkdownDescription: "AWS metrics destination. Set when publishing metrics to CloudWatch in the source account.",
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

func (r *SourceAccountMetricsConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SourceAccountMetricsConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceAccountMetricsConfigResourceModel

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

func (r *SourceAccountMetricsConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceAccountMetricsConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetSourceAccountMetricsConfig(ctx, data.SourceAccountId.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source account metrics config: %s", err))
		return
	}

	if !config.GetEnabled() {
		tflog.Debug(ctx, "Source account metrics are disabled; removing from state", map[string]interface{}{
			"source_account_id": data.SourceAccountId.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(config.GetSourceAccountId())
	data.SourceAccountId = types.StringValue(config.GetSourceAccountId())
	resp.Diagnostics.Append(sourceMetricsConfigToModel(config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAccountMetricsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SourceAccountMetricsConfigResourceModel

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

func (r *SourceAccountMetricsConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceAccountMetricsConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Disabling source account metrics config", map[string]interface{}{
		"source_account_id": data.SourceAccountId.ValueString(),
	})

	err := r.client.DisableSourceAccountMetricsConfig(ctx, data.SourceAccountId.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to disable source account metrics config: %s", err))
		return
	}
}

func (r *SourceAccountMetricsConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source_account_id"), req.ID)...)
}

func (r *SourceAccountMetricsConfigResource) apply(ctx context.Context, data *SourceAccountMetricsConfigResourceModel, diags *diag.Diagnostics) {
	enableReq := modelToEnableSourceMetricsRequest(data)

	tflog.Debug(ctx, "Enabling source account metrics config", map[string]interface{}{
		"source_account_id": data.SourceAccountId.ValueString(),
	})

	config, err := r.client.EnableSourceAccountMetricsConfig(ctx, data.SourceAccountId.ValueString(), enableReq)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to enable source account metrics config: %s", err))
		return
	}

	data.Id = types.StringValue(config.GetSourceAccountId())
	data.SourceAccountId = types.StringValue(config.GetSourceAccountId())
	diags.Append(sourceMetricsConfigToModel(config, data)...)
}

func modelToEnableSourceMetricsRequest(data *SourceAccountMetricsConfigResourceModel) externalEonSdkAPI.EnableSourceAccountMetricsConfigRequest {
	req := externalEonSdkAPI.EnableSourceAccountMetricsConfigRequest{}
	if data.Aws != nil {
		awsDest := externalEonSdkAPI.NewAwsAccountMetricsDestination()
		if !data.Aws.Region.IsNull() && !data.Aws.Region.IsUnknown() && data.Aws.Region.ValueString() != "" {
			awsDest.SetRegion(data.Aws.Region.ValueString())
		}
		req.SetAws(*awsDest)
	}
	return req
}

func sourceMetricsConfigToModel(config *externalEonSdkAPI.SourceAccountMetricsConfig, data *SourceAccountMetricsConfigResourceModel) diag.Diagnostics {
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
