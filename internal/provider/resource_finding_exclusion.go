package provider

import (
	"context"
	"encoding/json"
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
	Id         types.String `tfsdk:"id"`
	ResourceId types.String `tfsdk:"resource_id"`
	Value      types.String `tfsdk:"value"`
	Type       types.String `tfsdk:"type"`
	Detector   types.String `tfsdk:"detector"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func (r *FindingExclusionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_finding_exclusion"
}

func (r *FindingExclusionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Stops one detector from reporting findings for matching files, tables, or databases, on one resource or on every resource in the account. " +
			"An exclusion applies to scans that start after it is created or updated.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the finding exclusion.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the resource the exclusion applies to. Omit it to apply the exclusion to every resource in the account.",
				Optional:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "What the exclusion matches. For `PATH`, every file whose path starts with this value, so `/data/tmp` also matches `/data/tmp2/report.csv`. " +
					"Linux paths are compared case-sensitively. Windows paths are compared as the backup stores them, which is currently lowercase, so write Windows prefixes in lowercase (for example `c:/users/app/cache`). " +
					"For `TABLE` or `DATABASE`, the exact table or database name.",
				Required: true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "What the exclusion refers to. Supported values: `PATH` (file path prefix), `TABLE` (exact table name), `DATABASE` (exact database name).",
				Required:            true,
			},
			"detector": schema.StringAttribute{
				MarkdownDescription: "Detector whose findings this exclusion suppresses. Supported values: `RANSOMWARE_BEHAVIOR`, `DATA_ANOMALY`, `MALWARE`.",
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
		"value":    data.Value.ValueString(),
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

	tflog.Debug(ctx, "Updating finding exclusion", map[string]interface{}{"id": state.Id.ValueString()})

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

func findingExclusionPayload(data *FindingExclusionResourceModel) map[string]any {
	payload := map[string]any{
		"value":    data.Value.ValueString(),
		"type":     data.Type.ValueString(),
		"detector": data.Detector.ValueString(),
	}
	// A missing resourceId applies the exclusion to every resource in the account.
	if !data.ResourceId.IsNull() && !data.ResourceId.IsUnknown() && data.ResourceId.ValueString() != "" {
		payload["resourceId"] = data.ResourceId.ValueString()
	}
	return payload
}

func findingExclusionCreateRequest(data *FindingExclusionResourceModel) (externalEonSdkAPI.CreateFindingExclusionRequest, diag.Diagnostics) {
	var createReq externalEonSdkAPI.CreateFindingExclusionRequest
	diags := decodeFindingExclusionPayload(findingExclusionPayload(data), &createReq)
	return createReq, diags
}

func findingExclusionUpdateRequest(data *FindingExclusionResourceModel) (externalEonSdkAPI.UpdateFindingExclusionRequest, diag.Diagnostics) {
	var updateReq externalEonSdkAPI.UpdateFindingExclusionRequest
	diags := decodeFindingExclusionPayload(findingExclusionPayload(data), &updateReq)
	return updateReq, diags
}

func findingExclusionToState(exclusion *externalEonSdkAPI.FindingExclusion, data *FindingExclusionResourceModel) {
	mapped := findingExclusionStateFromAPI(exclusion)
	data.Id = mapped.ID
	data.ResourceId = mapped.ResourceID
	data.Value = mapped.Value
	data.Type = mapped.Type
	data.Detector = mapped.Detector
	data.UpdatedAt = mapped.UpdatedAt
}

func findingExclusionListItem(exclusion externalEonSdkAPI.FindingExclusion) FindingExclusionListItemModel {
	mapped := findingExclusionStateFromAPI(&exclusion)
	return FindingExclusionListItemModel{
		Id:         mapped.ID,
		ResourceId: mapped.ResourceID,
		Value:      mapped.Value,
		Type:       mapped.Type,
		Detector:   mapped.Detector,
		UpdatedAt:  mapped.UpdatedAt,
	}
}

type findingExclusionState struct {
	ID         types.String
	ResourceID types.String
	Value      types.String
	Type       types.String
	Detector   types.String
	UpdatedAt  types.String
}

func findingExclusionStateFromAPI(exclusion *externalEonSdkAPI.FindingExclusion) findingExclusionState {
	state := findingExclusionState{
		ID:         types.StringValue(exclusion.GetId()),
		ResourceID: types.StringNull(),
		Value:      types.StringValue(exclusion.GetValue()),
		Type:       types.StringValue(string(exclusion.GetType())),
		Detector:   types.StringValue(string(exclusion.GetDetector())),
		UpdatedAt:  types.StringValue(exclusion.GetUpdatedAt().UTC().Format(time.RFC3339)),
	}
	if id, ok := exclusion.GetResourceIdOk(); ok && id != nil && *id != "" {
		state.ResourceID = types.StringValue(*id)
	}
	return state
}

func decodeFindingExclusionPayload(payload map[string]any, target any) diag.Diagnostics {
	var diags diag.Diagnostics

	encoded, err := json.Marshal(payload)
	if err != nil {
		diags.AddError("Invalid Request", fmt.Sprintf("Unable to encode finding exclusion request: %s", err))
		return diags
	}

	if err := json.Unmarshal(encoded, target); err != nil {
		diags.AddError("Invalid Request", fmt.Sprintf("Unable to build finding exclusion request: %s", err))
	}

	return diags
}
