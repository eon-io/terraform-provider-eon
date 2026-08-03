package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Nested `group` conditions are unrolled into the schema, so the provider has to pick a maximum
// nesting depth up front. Three levels cover the posture controls the console can express.
const backupPostureControlMaxExpressionDepth = 3

// A rule the API omits is a rule it does not enforce, so the cross-copy flags are defaulted rather
// than left null: a config that omits them still matches the state refreshed from the API.
var backupPostureControlDefaultedRules = []string{"crossRegion", "crossAccount", "crossCloudProvider"}

var _ resource.Resource = &BackupPostureControlResource{}
var _ resource.ResourceWithImportState = &BackupPostureControlResource{}

func NewBackupPostureControlResource() resource.Resource {
	return &BackupPostureControlResource{}
}

type BackupPostureControlResource struct {
	client *client.EonClient
}

type BackupPostureControlResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Severity         types.String `tfsdk:"severity"`
	ResourceSelector types.Object `tfsdk:"resource_selector"`
	Rules            types.Object `tfsdk:"rules"`
}

// backupPostureControlCondition describes one resource-selector condition. `attr` and `values` are
// the Terraform attribute names; both map to their API counterparts by camel-casing.
type backupPostureControlCondition struct {
	attr      string
	values    string
	operators string
	desc      string
	// tagKeyValues carries key/value objects instead of plain strings.
	tagKeyValues bool
}

const (
	scalarOperators = "`IN`, `NOT_IN`"
	listOperators   = "`CONTAINS_ANY_OF`, `CONTAINS_NONE_OF`, `CONTAINS_ALL_OF`"
	stringOperators = "`IN`, `NOT_IN`, `CONTAINS`, `NOT_CONTAINS`, `STARTS_WITH`, `NOT_STARTS_WITH`, `ENDS_WITH`, `NOT_ENDS_WITH`"
)

var backupPostureControlConditions = []backupPostureControlCondition{
	{attr: "resource_type", values: "resource_types", operators: scalarOperators, desc: "Matches resources by resource type, for example `AWS_EC2` or `GCP_CLOUD_SQL`."},
	{attr: "environment", values: "environments", operators: scalarOperators, desc: "Matches resources by environment, for example `PROD` or `DEV`."},
	{attr: "cloud_provider", values: "cloud_providers", operators: scalarOperators, desc: "Matches resources by cloud provider, for example `AWS`, `AZURE` or `GCP`."},
	{attr: "account_id", values: "account_ids", operators: scalarOperators, desc: "Matches resources by the cloud account they live in."},
	{attr: "source_region", values: "regions", operators: scalarOperators, desc: "Matches resources by source region."},
	{attr: "vpc", values: "vpcs", operators: scalarOperators, desc: "Matches resources by VPC."},
	{attr: "subnets", values: "subnets", operators: listOperators, desc: "Matches resources by subnet."},
	{attr: "resource_group_name", values: "resource_group_names", operators: scalarOperators, desc: "Matches resources by resource group name (Azure)."},
	{attr: "resource_name", values: "resource_names", operators: stringOperators, desc: "Matches resources by resource name."},
	{attr: "resource_id", values: "resource_ids", operators: scalarOperators, desc: "Matches resources by Eon-assigned resource ID."},
	{attr: "apps", values: "apps", operators: listOperators, desc: "Matches resources by detected application."},
	{attr: "data_classes", values: "data_classes", operators: listOperators, desc: "Matches resources by detected data class."},
	{attr: "tag_keys", values: "tag_keys", operators: listOperators, desc: "Matches resources by cloud tag key."},
	{attr: "tag_key_values", values: "tag_key_values", operators: listOperators, desc: "Matches resources by cloud tag key and value.", tagKeyValues: true},
}

var backupPostureControlSchema = buildBackupPostureControlSchema()

func buildBackupPostureControlSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a backup posture control: the backup requirements a set of resources must satisfy, and the severity of the violations raised when they do not. " +
			"`resource_selector` picks the resources the control applies to, `rules` states the requirements. A rule you omit is not enforced.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the control.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the control.",
				Required:            true,
			},
			"severity": schema.StringAttribute{
				MarkdownDescription: "Severity assigned to violations of this control. Supported values: `HIGH`, `MEDIUM`, `LOW`.",
				Required:            true,
			},
			"resource_selector": schema.SingleNestedAttribute{
				MarkdownDescription: "Determines which resources this control applies to.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"resource_selection_mode": schema.StringAttribute{
						MarkdownDescription: "How the control selects resources. Supported values: `ALL`, `NONE`, `CONDITIONAL`. `CONDITIONAL` requires `expression`.",
						Required:            true,
					},
					"expression": schema.SingleNestedAttribute{
						MarkdownDescription: "Condition selecting the resources this control applies to. Set exactly one attribute; use `group` to combine conditions with `AND`/`OR`. Required when `resource_selection_mode` is `CONDITIONAL`, ignored otherwise.",
						Optional:            true,
						Attributes:          backupPostureControlExpressionAttributes(backupPostureControlMaxExpressionDepth),
					},
				},
			},
			"rules": schema.SingleNestedAttribute{
				MarkdownDescription: "The backup requirements a matching resource must satisfy. Every rule is optional: a rule you set is evaluated, a rule you omit is not enforced.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"minimum_retention": schema.ListNestedAttribute{
						MarkdownDescription: "Minimum retention required per backup frequency.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"frequency": schema.StringAttribute{
									MarkdownDescription: "The backup cadence this minimum retention applies to. Supported values: `DAILY`, `WEEKLY`, `MONTHLY`, `ANNUAL`.",
									Required:            true,
								},
								"minimum_retention": schema.Int64Attribute{
									MarkdownDescription: "The minimum backup retention period, in days.",
									Required:            true,
								},
							},
						},
					},
					"maximum_retention": schema.SingleNestedAttribute{
						MarkdownDescription: "Maximum retention a backup may keep before it is considered non-compliant.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"maximum_retention": schema.Int64Attribute{
								MarkdownDescription: "The maximum backup retention period, in days.",
								Required:            true,
							},
						},
					},
					"number_of_copies": schema.SingleNestedAttribute{
						MarkdownDescription: "Minimum number of backup copies that must exist.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"min_copies": schema.Int64Attribute{
								MarkdownDescription: "The minimum number of backup copies that must exist.",
								Required:            true,
							},
						},
					},
					"cross_region": schema.BoolAttribute{
						MarkdownDescription: "Whether at least one backup copy must be stored in a different region.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"cross_account": schema.BoolAttribute{
						MarkdownDescription: "Whether at least one backup copy must be stored in a different cloud account.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"cross_cloud_provider": schema.BoolAttribute{
						MarkdownDescription: "Whether at least one backup copy must be stored in a different cloud provider.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
				},
			},
		},
	}
}

func backupPostureControlExpressionAttributes(depth int) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(backupPostureControlConditions)+1)

	for _, condition := range backupPostureControlConditions {
		var values schema.Attribute
		if condition.tagKeyValues {
			values = schema.ListNestedAttribute{
				MarkdownDescription: "Tag key-value pairs to match. Omit `value` to match any value of the key.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":   schema.StringAttribute{MarkdownDescription: "Tag key.", Required: true},
						"value": schema.StringAttribute{MarkdownDescription: "Tag value.", Optional: true},
					},
				},
			}
		} else {
			values = schema.ListAttribute{
				MarkdownDescription: "Values to match.",
				ElementType:         types.StringType,
				Required:            true,
			}
		}

		attrs[condition.attr] = schema.SingleNestedAttribute{
			MarkdownDescription: condition.desc,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{MarkdownDescription: "Operator. Supported values: " + condition.operators + ".", Required: true},
				condition.values: values,
			},
		}
	}

	if depth > 1 {
		attrs["group"] = schema.SingleNestedAttribute{
			MarkdownDescription: fmt.Sprintf("Combines at least two conditions with a logical operator. Groups nest up to %d levels deep.", backupPostureControlMaxExpressionDepth),
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Logical operator. Supported values: `AND`, `OR`.",
					Required:            true,
				},
				"operands": schema.ListNestedAttribute{
					MarkdownDescription: "The expressions to combine with `operator`. At least two.",
					Required:            true,
					NestedObject:        schema.NestedAttributeObject{Attributes: backupPostureControlExpressionAttributes(depth - 1)},
				},
			},
		}
	}

	return attrs
}

func (r *BackupPostureControlResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_posture_control"
}

func (r *BackupPostureControlResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = backupPostureControlSchema
}

func (r *BackupPostureControlResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BackupPostureControlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diags := backupPostureControlPayload(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var createReq externalEonSdkAPI.CreateBackupPostureControlRequest
	if diags := decodePayload(payload, &createReq); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "Creating backup posture control", map[string]interface{}{"name": data.Name.ValueString()})

	control, err := r.client.CreateBackupPostureControl(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create backup posture control: %s", err))
		return
	}

	// Only the ID comes from the API: the rest of the state stays as planned, so an API that echoes
	// an omitted rule back as its zero value cannot fail the apply.
	data.Id = types.StringValue(control.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPostureControlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	control, err := r.client.GetBackupPostureControl(ctx, data.Id.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, "Backup posture control not found, removing from state", map[string]interface{}{"id": data.Id.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup posture control: %s", err))
		return
	}

	resp.Diagnostics.Append(backupPostureControlToState(ctx, control, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPostureControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BackupPostureControlResourceModel
	var state BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diags := backupPostureControlPayload(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var updateReq externalEonSdkAPI.UpdateBackupPostureControlRequest
	if diags := decodePayload(payload, &updateReq); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	control, err := r.client.UpdateBackupPostureControl(ctx, state.Id.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update backup posture control: %s", err))
		return
	}

	data.Id = types.StringValue(control.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPostureControlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteBackupPostureControl(ctx, data.Id.ValueString()); err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete backup posture control: %s", err))
		return
	}
}

func (r *BackupPostureControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// backupPostureControlPayload renders the plan as the JSON body the Eon API expects.
func backupPostureControlPayload(ctx context.Context, data *BackupPostureControlResourceModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	selector, d := tfValueToAPI(ctx, data.ResourceSelector)
	diags.Append(d...)
	rules, d := tfValueToAPI(ctx, data.Rules)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	return map[string]any{
		"name":             data.Name.ValueString(),
		"severity":         data.Severity.ValueString(),
		"resourceSelector": selector,
		"rules":            rules,
	}, diags
}

func backupPostureControlToState(ctx context.Context, control *externalEonSdkAPI.BackupPostureControl, data *BackupPostureControlResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	raw, err := json.Marshal(control)
	if err != nil {
		diags.AddError("Invalid API Response", fmt.Sprintf("Unable to encode backup posture control: %s", err))
		return diags
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		diags.AddError("Invalid API Response", fmt.Sprintf("Unable to decode backup posture control: %s", err))
		return diags
	}

	if rules, ok := decoded["rules"].(map[string]any); ok {
		for _, name := range backupPostureControlDefaultedRules {
			if _, set := rules[name]; !set {
				rules[name] = false
			}
		}
	}

	selector, d := apiValueToTF(decoded["resourceSelector"], backupPostureControlAttributeType("resource_selector"))
	diags.Append(d...)
	rules, d := apiValueToTF(decoded["rules"], backupPostureControlAttributeType("rules"))
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	data.Id = types.StringValue(control.Id)
	data.Name = types.StringValue(control.Name)
	data.Severity = types.StringValue(string(control.Severity))
	data.ResourceSelector = selector.(types.Object)
	data.Rules = rules.(types.Object)

	return diags
}

func backupPostureControlAttributeType(name string) attr.Type {
	return backupPostureControlSchema.Attributes[name].GetType()
}

// decodePayload feeds a rendered payload through the SDK request type, which validates the required
// fields and drops nothing the API does not accept.
func decodePayload(payload map[string]any, target any) diag.Diagnostics {
	var diags diag.Diagnostics

	encoded, err := json.Marshal(payload)
	if err != nil {
		diags.AddError("Invalid Request", fmt.Sprintf("Unable to encode backup posture control request: %s", err))
		return diags
	}

	if err := json.Unmarshal(encoded, target); err != nil {
		diags.AddError("Invalid Request", fmt.Sprintf("Unable to build backup posture control request: %s", err))
	}

	return diags
}

// tfValueToAPI walks a plan value and renders it as the API's JSON shape. Terraform attribute names
// are snake_case and the API's are camelCase, so every object key is converted on the way out.
func tfValueToAPI(ctx context.Context, value attr.Value) (any, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value == nil || value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	switch typed := value.(type) {
	case types.String:
		return typed.ValueString(), diags
	case types.Bool:
		return typed.ValueBool(), diags
	case types.Int64:
		return typed.ValueInt64(), diags
	case types.Object:
		rendered := make(map[string]any)
		for name, nested := range typed.Attributes() {
			converted, d := tfValueToAPI(ctx, nested)
			diags.Append(d...)
			if converted == nil {
				continue
			}
			rendered[camelCase(name)] = converted
		}
		return rendered, diags
	case types.List:
		rendered := make([]any, 0, len(typed.Elements()))
		for _, element := range typed.Elements() {
			converted, d := tfValueToAPI(ctx, element)
			diags.Append(d...)
			rendered = append(rendered, converted)
		}
		return rendered, diags
	default:
		diags.AddError("Unsupported Attribute Type", fmt.Sprintf("Cannot render Terraform value of type %T as an Eon API payload", value))
		return nil, diags
	}
}

// apiValueToTF is the inverse of tfValueToAPI: it rebuilds Terraform state from a decoded API
// response, taking the target types from the resource schema so an imported control is complete.
func apiValueToTF(raw any, target attr.Type) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch typed := target.(type) {
	case basetypes.StringType:
		if raw == nil {
			return types.StringNull(), diags
		}
		value, ok := raw.(string)
		if !ok {
			return nil, typeMismatch(raw, "string")
		}
		return types.StringValue(value), diags
	case basetypes.BoolType:
		if raw == nil {
			return types.BoolNull(), diags
		}
		value, ok := raw.(bool)
		if !ok {
			return nil, typeMismatch(raw, "bool")
		}
		return types.BoolValue(value), diags
	case basetypes.Int64Type:
		if raw == nil {
			return types.Int64Null(), diags
		}
		value, ok := raw.(float64)
		if !ok {
			return nil, typeMismatch(raw, "number")
		}
		return types.Int64Value(int64(value)), diags
	case basetypes.ObjectType:
		if raw == nil {
			return types.ObjectNull(typed.AttrTypes), diags
		}
		object, ok := raw.(map[string]any)
		if !ok {
			return nil, typeMismatch(raw, "object")
		}
		attributes := make(map[string]attr.Value, len(typed.AttrTypes))
		for name, attrType := range typed.AttrTypes {
			converted, d := apiValueToTF(object[camelCase(name)], attrType)
			diags.Append(d...)
			if d.HasError() {
				continue
			}
			attributes[name] = converted
		}
		if diags.HasError() {
			return nil, diags
		}
		value, d := types.ObjectValue(typed.AttrTypes, attributes)
		diags.Append(d...)
		return value, diags
	case basetypes.ListType:
		items, ok := raw.([]any)
		// The API drops a rule it does not enforce, and returns an empty list for some of them.
		// Both mean "unset", and null is the shape Terraform config uses for that.
		if raw == nil || (ok && len(items) == 0) {
			return types.ListNull(typed.ElemType), diags
		}
		if !ok {
			return nil, typeMismatch(raw, "list")
		}
		elements := make([]attr.Value, 0, len(items))
		for _, item := range items {
			converted, d := apiValueToTF(item, typed.ElemType)
			diags.Append(d...)
			if d.HasError() {
				continue
			}
			elements = append(elements, converted)
		}
		if diags.HasError() {
			return nil, diags
		}
		value, d := types.ListValue(typed.ElemType, elements)
		diags.Append(d...)
		return value, diags
	default:
		diags.AddError("Unsupported Attribute Type", fmt.Sprintf("Cannot read an Eon API value into a Terraform attribute of type %T", target))
		return nil, diags
	}
}

func typeMismatch(raw any, expected string) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.AddError("Invalid API Response", fmt.Sprintf("Expected %s in the backup posture control response, got %T", expected, raw))
	return diags
}

func camelCase(name string) string {
	parts := strings.Split(name, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}
