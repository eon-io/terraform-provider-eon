package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

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

var _ resource.Resource = &FindingExclusionResource{}
var _ resource.ResourceWithImportState = &FindingExclusionResource{}

func NewFindingExclusionResource() resource.Resource {
	return &FindingExclusionResource{}
}

type FindingExclusionResource struct {
	client *client.EonClient
}

type FindingExclusionResourceModel struct {
	Id                 types.String `tfsdk:"id"`
	Scope              types.String `tfsdk:"scope"`
	ProviderResourceId types.String `tfsdk:"provider_resource_id"`
	Value              types.String `tfsdk:"value"`
	Type               types.String `tfsdk:"type"`
	Detector           types.String `tfsdk:"detector"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *FindingExclusionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_finding_exclusion"
}

func (r *FindingExclusionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a finding exclusion that stops one detector from reporting findings for matching files, tables, or databases, on one resource or on every resource in the account. An exclusion applies to scans that start after it is created or updated.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Finding exclusion ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Where the exclusion applies. `RESOURCE` applies it to the resource named by `provider_resource_id`. `ACCOUNT` applies it to every resource in the account. Supported values: `RESOURCE`, `ACCOUNT`.",
				Required:            true,
			},
			"provider_resource_id": schema.StringAttribute{
				MarkdownDescription: "Cloud provider ID of the resource the exclusion applies to, such as an EC2 instance ID or an Azure resource ID. Required when `scope` is `RESOURCE`, and must be omitted when `scope` is `ACCOUNT`.",
				Optional:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "What the exclusion matches. For `PATH`, every file whose path starts with this value. Linux paths are compared case-sensitively. Windows paths are compared as the backup stores them, which is currently lowercase, so write Windows prefixes in lowercase (for example `c:/users/app/cache`). For `TABLE` or `DATABASE`, the exact table or database name.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "What the exclusion matches. Supported values: `PATH`, `TABLE`, `DATABASE`.",
				Required:            true,
			},
			"detector": schema.StringAttribute{
				MarkdownDescription: "Detector whose findings the exclusion suppresses. Supported values: `RANSOMWARE_BEHAVIOR`, `DATA_ANOMALY`, `MALWARE`.",
				Required:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the exclusion was created or last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *FindingExclusionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FindingExclusionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FindingExclusionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := findingExclusionCreateRequest(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating finding exclusion", map[string]interface{}{
		"scope":    data.Scope.ValueString(),
		"type":     data.Type.ValueString(),
		"detector": data.Detector.ValueString(),
	})

	exclusion, err := r.client.CreateFindingExclusion(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create finding exclusion: %s", err))
		return
	}

	findingExclusionToState(exclusion, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FindingExclusionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FindingExclusionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exclusion, err := r.client.GetFindingExclusion(ctx, data.Id.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, "Finding exclusion not found, removing from state", map[string]interface{}{"id": data.Id.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read finding exclusion: %s", err))
		return
	}

	findingExclusionToState(exclusion, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FindingExclusionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FindingExclusionResourceModel
	var state FindingExclusionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := findingExclusionUpdateRequest(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exclusion, err := r.client.UpdateFindingExclusion(ctx, state.Id.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update finding exclusion: %s", err))
		return
	}

	findingExclusionToState(exclusion, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FindingExclusionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FindingExclusionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteFindingExclusion(ctx, data.Id.ValueString()); err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete finding exclusion: %s", err))
		return
	}
}

func (r *FindingExclusionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type findingExclusionWriteFields struct {
	scope              externalEonSdkAPI.FindingExclusionScope
	value              string
	objectType         externalEonSdkAPI.FindingObjectType
	detector           externalEonSdkAPI.FindingExclusionDetectorType
	providerResourceID *string
}

func findingExclusionWriteFieldsFromModel(data *FindingExclusionResourceModel) (findingExclusionWriteFields, diag.Diagnostics) {
	var diags diag.Diagnostics
	var fields findingExclusionWriteFields

	scope, err := externalEonSdkAPI.NewFindingExclusionScopeFromValue(data.Scope.ValueString())
	if err != nil {
		diags.AddError("Invalid Attribute", fmt.Sprintf("scope: %s", err))
		return fields, diags
	}
	objectType, err := externalEonSdkAPI.NewFindingObjectTypeFromValue(data.Type.ValueString())
	if err != nil {
		diags.AddError("Invalid Attribute", fmt.Sprintf("type: %s", err))
		return fields, diags
	}
	detector, err := externalEonSdkAPI.NewFindingExclusionDetectorTypeFromValue(data.Detector.ValueString())
	if err != nil {
		diags.AddError("Invalid Attribute", fmt.Sprintf("detector: %s", err))
		return fields, diags
	}

	hasProviderID := !data.ProviderResourceId.IsNull() && !data.ProviderResourceId.IsUnknown() && data.ProviderResourceId.ValueString() != ""
	if *scope == externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE && !hasProviderID {
		diags.AddError("Invalid Attribute", "provider_resource_id is required when scope is RESOURCE")
		return fields, diags
	}
	if *scope == externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT && hasProviderID {
		diags.AddError("Invalid Attribute", "provider_resource_id must be omitted when scope is ACCOUNT")
		return fields, diags
	}

	fields.scope = *scope
	fields.value = data.Value.ValueString()
	fields.objectType = *objectType
	fields.detector = *detector
	if hasProviderID {
		id := data.ProviderResourceId.ValueString()
		fields.providerResourceID = &id
	}
	return fields, diags
}

func findingExclusionCreateRequest(data *FindingExclusionResourceModel) (externalEonSdkAPI.CreateFindingExclusionRequest, diag.Diagnostics) {
	fields, diags := findingExclusionWriteFieldsFromModel(data)
	if diags.HasError() {
		return externalEonSdkAPI.CreateFindingExclusionRequest{}, diags
	}

	req := externalEonSdkAPI.NewCreateFindingExclusionRequest(fields.scope, fields.value, fields.objectType, fields.detector)
	if fields.providerResourceID != nil {
		req.SetProviderResourceId(*fields.providerResourceID)
	}
	return *req, diags
}

func findingExclusionUpdateRequest(data *FindingExclusionResourceModel) (externalEonSdkAPI.UpdateFindingExclusionRequest, diag.Diagnostics) {
	fields, diags := findingExclusionWriteFieldsFromModel(data)
	if diags.HasError() {
		return externalEonSdkAPI.UpdateFindingExclusionRequest{}, diags
	}

	req := externalEonSdkAPI.NewUpdateFindingExclusionRequest(fields.scope, fields.value, fields.objectType, fields.detector)
	if fields.providerResourceID != nil {
		req.SetProviderResourceId(*fields.providerResourceID)
	}
	return *req, diags
}

func findingExclusionToState(exclusion *externalEonSdkAPI.FindingExclusion, data *FindingExclusionResourceModel) {
	data.Id = types.StringValue(exclusion.GetId())
	data.Scope = types.StringValue(string(exclusion.GetScope()))
	data.Value = types.StringValue(exclusion.GetValue())
	data.Type = types.StringValue(string(exclusion.GetType()))
	data.Detector = types.StringValue(string(exclusion.GetDetector()))
	data.UpdatedAt = types.StringValue(exclusion.GetUpdatedAt().UTC().Format(time.RFC3339))
	if id, ok := exclusion.GetProviderResourceIdOk(); ok && id != nil && *id != "" {
		data.ProviderResourceId = types.StringValue(*id)
	} else {
		data.ProviderResourceId = types.StringNull()
	}
}
