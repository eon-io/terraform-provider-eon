package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ActionApprovalRulesDataSource{}

func NewActionApprovalRulesDataSource() datasource.DataSource {
	return &ActionApprovalRulesDataSource{}
}

type ActionApprovalRulesDataSource struct {
	client *client.EonClient
}

type ActionApprovalRulesDataSourceModel struct {
	Rules []ActionApprovalRuleListItemModel `tfsdk:"rules"`
}

type ActionApprovalRuleListItemModel struct {
	Id                   types.String `tfsdk:"id"`
	Operation            types.String `tfsdk:"operation"`
	RequiredApprovals    types.Int64  `tfsdk:"required_approvals"`
	ApprovalWindowHours  types.Int64  `tfsdk:"approval_window_hours"`
	ExecutionWindowHours types.Int64  `tfsdk:"execution_window_hours"`
	Description          types.String `tfsdk:"description"`
}

func (d *ActionApprovalRulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_approval_rules"
}

func (d *ActionApprovalRulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of action approval rules in the Eon project.",
		Attributes: map[string]schema.Attribute{
			"rules": schema.ListNestedAttribute{
				MarkdownDescription: "List of action approval rules.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned ID of the action approval rule.",
							Computed:            true,
						},
						"operation": schema.StringAttribute{
							MarkdownDescription: "Action protected by this rule.",
							Computed:            true,
						},
						"required_approvals": schema.Int64Attribute{
							MarkdownDescription: "Number of approvals required before the action can be executed.",
							Computed:            true,
						},
						"approval_window_hours": schema.Int64Attribute{
							MarkdownDescription: "Hours the request stays open for approval before expiring.",
							Computed:            true,
						},
						"execution_window_hours": schema.Int64Attribute{
							MarkdownDescription: "Hours after approval during which the approved action can be executed.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Optional description explaining the purpose of this rule.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *ActionApprovalRulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *ActionApprovalRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActionApprovalRulesDataSourceModel

	rules, err := d.client.ListActionApprovalRules(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read action approval rules: %s", err))
		return
	}

	for _, rule := range rules {
		item := ActionApprovalRuleListItemModel{
			Id:                   types.StringValue(rule.GetId()),
			Operation:            types.StringValue(string(rule.GetOperation())),
			RequiredApprovals:    types.Int64Value(int64(rule.GetRequiredApprovals())),
			ApprovalWindowHours:  types.Int64Value(int64(rule.GetApprovalWindowHours())),
			ExecutionWindowHours: types.Int64Value(int64(rule.GetExecutionWindowHours())),
			Description:          types.StringNull(),
		}
		if rule.HasDescription() {
			item.Description = types.StringValue(rule.GetDescription())
		}
		data.Rules = append(data.Rules, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
