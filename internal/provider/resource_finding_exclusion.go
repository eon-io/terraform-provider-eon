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
		MarkdownDescription: "Manages a finding exclusion: stops one threat detector from reporting findings for matching files, tables or databases, on one resource or on every resource in the account. An exclusion applies to scans that start after it's created or updated.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Finding exclusion ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the resource the exclusion applies to. Omit it to apply the exclusion to every resource in the account.",
				Optional:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "What the exclusion matches, depending on `type`. For `PATH`, every file whose path starts with this value, so `/data/tmp` also matches `/data/tmp2/report.csv`; Linux paths are compared case-sensitively, and Windows paths as the backup stores them, which is currently lowercase (write them in lowercase, for example `c:/users/app/cache`). For `TABLE` or `DATABASE`, the exact table or database name.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "What `value` refers to: `PATH`, `TABLE` or `DATABASE`.",
				Required:            true,
			},
			"detector": schema.StringAttribute{
				MarkdownDescription: "Detector whose findings to suppress: `RANSOMWARE_BEHAVIOR`, `DATA_ANOMALY` or `MALWARE`. `MALWARE` can only exclude paths: databases aren't scanned for malware.",
				Required:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the exclusion was created or last updated, in RFC 3339 format.",
				Computed:            true,
			},
		},
	}
}

func (r *FindingExclusionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
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
		if isNotFound(err) {
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
	var plan FindingExclusionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := findingExclusionUpdateRequest(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating finding exclusion", map[string]interface{}{"id": plan.Id.ValueString()})

	exclusion, err := r.client.UpdateFindingExclusion(ctx, plan.Id.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update finding exclusion: %s", err))
		return
	}

	findingExclusionToState(exclusion, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FindingExclusionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FindingExclusionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting finding exclusion", map[string]interface{}{"id": data.Id.ValueString()})

	if err := r.client.DeleteFindingExclusion(ctx, data.Id.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete finding exclusion: %s", err))
	}
}

func (r *FindingExclusionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func isNotFound(err error) bool {
	var apiErr *client.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// findingExclusionFields checks the enum attributes before calling the API, so a typo is reported on the attribute with
// its allowed values rather than as an API error.
func findingExclusionFields(data *FindingExclusionResourceModel) (externalEonSdkAPI.FindingObjectType, externalEonSdkAPI.FindingExclusionDetectorType, diag.Diagnostics) {
	var diags diag.Diagnostics
	objectType, err := externalEonSdkAPI.NewFindingObjectTypeFromValue(data.Type.ValueString())
	if err != nil {
		diags.AddAttributeError(path.Root("type"), "Invalid finding exclusion type",
			fmt.Sprintf("%q is not one of %v", data.Type.ValueString(), externalEonSdkAPI.AllowedFindingObjectTypeEnumValues))
	}
	detector, err := externalEonSdkAPI.NewFindingExclusionDetectorTypeFromValue(data.Detector.ValueString())
	if err != nil {
		diags.AddAttributeError(path.Root("detector"), "Invalid finding exclusion detector",
			fmt.Sprintf("%q is not one of %v", data.Detector.ValueString(), externalEonSdkAPI.AllowedFindingExclusionDetectorTypeEnumValues))
	}
	if diags.HasError() {
		return "", "", diags
	}
	return *objectType, *detector, diags
}

func findingExclusionCreateRequest(data *FindingExclusionResourceModel) (externalEonSdkAPI.CreateFindingExclusionRequest, diag.Diagnostics) {
	objectType, detector, diags := findingExclusionFields(data)
	if diags.HasError() {
		return externalEonSdkAPI.CreateFindingExclusionRequest{}, diags
	}
	req := externalEonSdkAPI.NewCreateFindingExclusionRequest(data.Value.ValueString(), objectType, detector)
	if resourceId := data.ResourceId.ValueString(); resourceId != "" {
		req.SetResourceId(resourceId)
	}
	return *req, diags
}

func findingExclusionUpdateRequest(data *FindingExclusionResourceModel) (externalEonSdkAPI.UpdateFindingExclusionRequest, diag.Diagnostics) {
	objectType, detector, diags := findingExclusionFields(data)
	if diags.HasError() {
		return externalEonSdkAPI.UpdateFindingExclusionRequest{}, diags
	}
	req := externalEonSdkAPI.NewUpdateFindingExclusionRequest(data.Value.ValueString(), objectType, detector)
	if resourceId := data.ResourceId.ValueString(); resourceId != "" {
		req.SetResourceId(resourceId)
	}
	return *req, diags
}

func findingExclusionToState(exclusion *externalEonSdkAPI.FindingExclusion, data *FindingExclusionResourceModel) {
	data.Id = types.StringValue(exclusion.GetId())
	if resourceId, ok := exclusion.GetResourceIdOk(); ok && resourceId != nil && *resourceId != "" {
		data.ResourceId = types.StringValue(*resourceId)
	} else {
		data.ResourceId = types.StringNull()
	}
	data.Value = types.StringValue(exclusion.GetValue())
	data.Type = types.StringValue(string(exclusion.GetType()))
	data.Detector = types.StringValue(string(exclusion.GetDetector()))
	data.UpdatedAt = types.StringValue(exclusion.GetUpdatedAt().Format(time.RFC3339))
}
