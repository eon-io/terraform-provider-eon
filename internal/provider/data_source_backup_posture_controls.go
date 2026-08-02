package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BackupPostureControlsDataSource{}

func NewBackupPostureControlsDataSource() datasource.DataSource {
	return &BackupPostureControlsDataSource{}
}

type BackupPostureControlsDataSource struct {
	client *client.EonClient
}

type BackupPostureControlsDataSourceModel struct {
	Controls []BackupPostureControlModel `tfsdk:"controls"`
}

type BackupPostureControlModel struct {
	Id                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Severity              types.String `tfsdk:"severity"`
	ResourceSelectionMode types.String `tfsdk:"resource_selection_mode"`
}

func (d *BackupPostureControlsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_posture_controls"
}

func (d *BackupPostureControlsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of backup posture controls in the Eon project.",
		Attributes: map[string]schema.Attribute{
			"controls": schema.ListNestedAttribute{
				MarkdownDescription: "List of backup posture controls.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Backup posture control ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Backup posture control display name.",
							Computed:            true,
						},
						"severity": schema.StringAttribute{
							MarkdownDescription: "Severity assigned to the control. Possible values: `HIGH`, `MEDIUM`, `LOW`.",
							Computed:            true,
						},
						"resource_selection_mode": schema.StringAttribute{
							MarkdownDescription: "Mode that determines how resources are selected for the control. Possible values: `ALL`, `NONE`, `CONDITIONAL`.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *BackupPostureControlsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BackupPostureControlsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BackupPostureControlsDataSourceModel
	data.Controls = []BackupPostureControlModel{}

	controls, err := d.client.ListBackupPostureControls(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup posture controls: %s", err))
		return
	}

	for _, control := range controls {
		selector := control.GetResourceSelector()
		data.Controls = append(data.Controls, BackupPostureControlModel{
			Id:                    types.StringValue(control.GetId()),
			Name:                  types.StringValue(control.GetName()),
			Severity:              types.StringValue(string(control.GetSeverity())),
			ResourceSelectionMode: types.StringValue(string(selector.ResourceSelectionMode)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
