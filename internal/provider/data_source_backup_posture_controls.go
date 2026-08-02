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
							MarkdownDescription: "Severity reported when a resource violates the control. Possible values: `LOW`, `MEDIUM`, `HIGH`.",
							Computed:            true,
						},
						"resource_selection_mode": schema.StringAttribute{
							MarkdownDescription: "Mode that determines how resources are selected for the control. Possible values: `ALL`, `CONDITIONAL`.",
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

func (d *BackupPostureControlsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BackupPostureControlsDataSourceModel

	controls, err := d.client.ListBackupPostureControls(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup posture controls: %s", err))
		return
	}

	for _, control := range controls {
		selectionMode := types.StringNull()
		if selector := control.ResourceSelector.Get(); selector != nil {
			selectionMode = types.StringValue(string(selector.ResourceSelectionMode))
		}

		data.Controls = append(data.Controls, BackupPostureControlModel{
			Id:                    types.StringValue(control.Id),
			Name:                  types.StringValue(control.Name),
			Severity:              types.StringValue(string(control.Severity)),
			ResourceSelectionMode: selectionMode,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
