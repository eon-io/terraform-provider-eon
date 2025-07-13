package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BackupPoliciesDataSource{}

func NewBackupPoliciesDataSource() datasource.DataSource {
	return &BackupPoliciesDataSource{}
}

type BackupPoliciesDataSource struct {
	client *client.EonClient
}

type BackupPoliciesDataSourceModel struct {
	Policies []BackupPolicyModel `tfsdk:"policies"`
}

type BackupPolicyModel struct {
	Id                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Enabled                   types.Bool   `tfsdk:"enabled"`
	BackupPolicyType          types.String `tfsdk:"backup_policy_type"`
	ResourceSelectionMode     types.String `tfsdk:"resource_selection_mode"`
	ResourceInclusionOverride types.List   `tfsdk:"resource_inclusion_override"`
	ResourceExclusionOverride types.List   `tfsdk:"resource_exclusion_override"`
}

func (d *BackupPoliciesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_policies"
}

func (d *BackupPoliciesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of backup policies in the Eon project.",
		Attributes: map[string]schema.Attribute{
			"policies": schema.ListNestedAttribute{
				MarkdownDescription: "List of backup policies.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Backup policy ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Backup policy display name.",
							Computed:            true,
						},
						"enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether the backup policy is enabled.",
							Computed:            true,
						},
						"backup_policy_type": schema.StringAttribute{
							MarkdownDescription: "The type of the policy. Possible values: `UNSPECIFIED`, `STANDARD`, `HIGH_FREQUENCY`.",
							Computed:            true,
						},
						"resource_selection_mode": schema.StringAttribute{
							MarkdownDescription: "Mode that determines how resources are selected for inclusion in the backup policy. To include or exclude all resources from the policy, set to `ALL` or `NONE`, respectively. For conditional selection, set to `CONDITIONAL`. Possible values: `ALL`, `NONE`, `CONDITIONAL`.",
							Computed:            true,
						},
						"resource_inclusion_override": schema.ListAttribute{
							MarkdownDescription: "List of cloud-provider-assigned resource IDs to include in the backup policy, regardless of whether they're excluded by `resource_selection_mode`.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"resource_exclusion_override": schema.ListAttribute{
							MarkdownDescription: "List of cloud-provider-assigned resource IDs to exclude from the backup policy, regardless of whether they're included by `resource_selection_mode`.",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *BackupPoliciesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *BackupPoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BackupPoliciesDataSourceModel

	policies, err := d.client.ListBackupPolicies(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup policies: %s", err))
		return
	}

	for _, policy := range policies {
		var inclusionOverride types.List
		if policy.ResourceSelector.ResourceInclusionOverride != nil {
			inclusionList, diags := types.ListValueFrom(ctx, types.StringType, policy.ResourceSelector.ResourceInclusionOverride)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			inclusionOverride = inclusionList
		} else {
			inclusionOverride = types.ListNull(types.StringType)
		}

		var exclusionOverride types.List
		if policy.ResourceSelector.ResourceExclusionOverride != nil {
			exclusionList, diags := types.ListValueFrom(ctx, types.StringType, policy.ResourceSelector.ResourceExclusionOverride)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			exclusionOverride = exclusionList
		} else {
			exclusionOverride = types.ListNull(types.StringType)
		}

		policyModel := BackupPolicyModel{
			Id:                        types.StringValue(policy.Id),
			Name:                      types.StringValue(policy.Name),
			Enabled:                   types.BoolValue(policy.Enabled),
			BackupPolicyType:          types.StringValue(string(policy.BackupPlan.BackupPolicyType)),
			ResourceSelectionMode:     types.StringValue(string(policy.ResourceSelector.ResourceSelectionMode)),
			ResourceInclusionOverride: inclusionOverride,
			ResourceExclusionOverride: exclusionOverride,
		}

		data.Policies = append(data.Policies, policyModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
