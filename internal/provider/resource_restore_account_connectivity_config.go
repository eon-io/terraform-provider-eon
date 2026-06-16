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

var _ resource.Resource = &RestoreAccountConnectivityConfigResource{}
var _ resource.ResourceWithImportState = &RestoreAccountConnectivityConfigResource{}

func NewRestoreAccountConnectivityConfigResource() resource.Resource {
	return &RestoreAccountConnectivityConfigResource{}
}

type RestoreAccountConnectivityConfigResource struct {
	client *client.EonClient
}

// RestoreAccountConnectivityConfigResourceModel is the Terraform model for a restore account's
// connectivity configuration. The configuration is a singleton per restore account, so the
// resource id mirrors restore_account_id.
type RestoreAccountConnectivityConfigResourceModel struct {
	Id               types.String                `tfsdk:"id"`
	RestoreAccountId types.String                `tfsdk:"restore_account_id"`
	Aws              *awsConnectivityConfigModel `tfsdk:"aws"`
	Gcp              *gcpConnectivityConfigModel `tfsdk:"gcp"`
}

type awsConnectivityConfigModel struct {
	VpcConfigs []awsVpcConnectivityConfigModel `tfsdk:"vpc_configs"`
}

type awsVpcConnectivityConfigModel struct {
	Region                     types.String                      `tfsdk:"region"`
	Vpc                        types.String                      `tfsdk:"vpc"`
	SubnetsPerAvailabilityZone []subnetPerAvailabilityZoneModel  `tfsdk:"subnets_per_availability_zone"`
	SecurityGroups             *resourceTypeToSecurityGroupModel `tfsdk:"security_groups"`
}

type subnetPerAvailabilityZoneModel struct {
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	SubnetId         types.String `tfsdk:"subnet_id"`
}

type resourceTypeToSecurityGroupModel struct {
	RestoreServer       types.List `tfsdk:"restore_server"`
	RestoredRdsInstance types.List `tfsdk:"restored_rds_instance"`
}

type gcpConnectivityConfigModel struct {
	NetworkConfigs []gcpNetworkConnectivityConfigModel `tfsdk:"network_configs"`
}

type gcpNetworkConnectivityConfigModel struct {
	Network            types.String              `tfsdk:"network"`
	SubnetsPerRegion   []gcpSubnetPerRegionModel `tfsdk:"subnets_per_region"`
	IsSharedVpc        types.Bool                `tfsdk:"is_shared_vpc"`
	NetworkHostProject types.String              `tfsdk:"network_host_project"`
}

type gcpSubnetPerRegionModel struct {
	Region     types.String `tfsdk:"region"`
	SubnetName types.String `tfsdk:"subnet_name"`
}

func (r *RestoreAccountConnectivityConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_connectivity_config"
}

func (r *RestoreAccountConnectivityConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the connectivity configuration of a restore account. The connectivity configuration controls the networks (VPCs/subnets/security groups) Eon uses when restoring resources into the account. The configuration is a singleton per restore account; deleting this resource reverts the account to its default connectivity settings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account. Mirrors `restore_account_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account whose connectivity configuration is managed.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
		Blocks: map[string]schema.Block{
			"aws": schema.SingleNestedBlock{
				MarkdownDescription: "AWS connectivity configuration. Set when the restore account is an AWS account.",
				Blocks: map[string]schema.Block{
					"vpc_configs": schema.ListNestedBlock{
						MarkdownDescription: "VPCs to configure.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"region": schema.StringAttribute{
									MarkdownDescription: "VPC region.",
									Required:            true,
								},
								"vpc": schema.StringAttribute{
									MarkdownDescription: "VPC ID.",
									Required:            true,
								},
							},
							Blocks: map[string]schema.Block{
								"subnets_per_availability_zone": schema.ListNestedBlock{
									MarkdownDescription: "Subnets to configure for availability zones in the VPC. For availability zones not specified, Eon attempts to use the default subnet.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"availability_zone": schema.StringAttribute{
												MarkdownDescription: "Availability zone.",
												Required:            true,
											},
											"subnet_id": schema.StringAttribute{
												MarkdownDescription: "Subnet ID to use.",
												Required:            true,
											},
										},
									},
								},
								"security_groups": schema.SingleNestedBlock{
									MarkdownDescription: "Security groups to use for the restore server and restored RDS instances.",
									Attributes: map[string]schema.Attribute{
										"restore_server": schema.ListAttribute{
											MarkdownDescription: "Security group to use for the restore server. If not specified, Eon attempts to use the default security group. Currently, a single security group is supported.",
											ElementType:         types.StringType,
											Optional:            true,
										},
										"restored_rds_instance": schema.ListAttribute{
											MarkdownDescription: "Security group to use for restored RDS instances. If not specified, Eon attempts to use the default security group. Currently, a single security group is supported.",
											ElementType:         types.StringType,
											Optional:            true,
										},
									},
								},
							},
						},
					},
				},
			},
			"gcp": schema.SingleNestedBlock{
				MarkdownDescription: "GCP connectivity configuration. Set when the restore account is a GCP account.",
				Blocks: map[string]schema.Block{
					"network_configs": schema.ListNestedBlock{
						MarkdownDescription: "Networks to configure.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"network": schema.StringAttribute{
									MarkdownDescription: "Network name.",
									Required:            true,
								},
								"is_shared_vpc": schema.BoolAttribute{
									MarkdownDescription: "Indicates whether the VPC network is a shared VPC. If true, `network_host_project` must be specified.",
									Required:            true,
								},
								"network_host_project": schema.StringAttribute{
									MarkdownDescription: "ID of the project that hosts the VPC network. Applicable for shared VPC networks.",
									Required:            true,
								},
							},
							Blocks: map[string]schema.Block{
								"subnets_per_region": schema.ListNestedBlock{
									MarkdownDescription: "Subnets to configure for regions in the network. For regions not specified, Eon attempts to use the default subnet.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"region": schema.StringAttribute{
												MarkdownDescription: "Region.",
												Required:            true,
											},
											"subnet_name": schema.StringAttribute{
												MarkdownDescription: "Subnet name.",
												Required:            true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *RestoreAccountConnectivityConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RestoreAccountConnectivityConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RestoreAccountConnectivityConfigResourceModel

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

func (r *RestoreAccountConnectivityConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RestoreAccountConnectivityConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetRestoreAccountConnectivityConfig(ctx, data.RestoreAccountId.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			// Restore account no longer exists; drop the resource from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account connectivity config: %s", err))
		return
	}

	data.Id = data.RestoreAccountId
	resp.Diagnostics.Append(connectivityConfigToModel(ctx, config, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreAccountConnectivityConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RestoreAccountConnectivityConfigResourceModel

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

func (r *RestoreAccountConnectivityConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RestoreAccountConnectivityConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting restore account connectivity config", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
	})

	err := r.client.DeleteRestoreAccountConnectivityConfig(ctx, data.RestoreAccountId.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			// Already gone — nothing to do.
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete restore account connectivity config: %s", err))
		return
	}
}

func (r *RestoreAccountConnectivityConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("restore_account_id"), req.ID)...)
}

// apply pushes the configuration in data to the API and updates the computed id in place.
func (r *RestoreAccountConnectivityConfigResource) apply(ctx context.Context, data *RestoreAccountConnectivityConfigResourceModel, diags *diag.Diagnostics) {
	updateReq, d := modelToUpdateRequest(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating restore account connectivity config", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
	})

	_, err := r.client.UpdateRestoreAccountConnectivityConfig(ctx, data.RestoreAccountId.ValueString(), updateReq)
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Unable to update restore account connectivity config: %s", err))
		return
	}

	data.Id = data.RestoreAccountId
}

// modelToUpdateRequest converts the Terraform model into the SDK update request.
func modelToUpdateRequest(ctx context.Context, data *RestoreAccountConnectivityConfigResourceModel) (externalEonSdkAPI.UpdateRestoreAccountConnectivityConfigRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := externalEonSdkAPI.UpdateRestoreAccountConnectivityConfigRequest{}

	if data.Aws != nil {
		vpcConfigs := make([]externalEonSdkAPI.AwsVpcConnectivityConfig, 0, len(data.Aws.VpcConfigs))
		for _, v := range data.Aws.VpcConfigs {
			vpc := externalEonSdkAPI.AwsVpcConnectivityConfig{
				Region: v.Region.ValueString(),
				Vpc:    v.Vpc.ValueString(),
			}
			for _, s := range v.SubnetsPerAvailabilityZone {
				vpc.SubnetsPerAvailabilityZone = append(vpc.SubnetsPerAvailabilityZone, externalEonSdkAPI.SubnetPerAvailabilityZone{
					AvailabilityZone: s.AvailabilityZone.ValueString(),
					SubnetId:         s.SubnetId.ValueString(),
				})
			}
			if v.SecurityGroups != nil {
				restoreServer, d := listOfStringFromModel(ctx, v.SecurityGroups.RestoreServer)
				diags.Append(d...)
				restoredRds, d := listOfStringFromModel(ctx, v.SecurityGroups.RestoredRdsInstance)
				diags.Append(d...)
				vpc.SecurityGroups = &externalEonSdkAPI.ResourceTypeToSecurityGroup{
					RestoreServer:       restoreServer,
					RestoredRdsInstance: restoredRds,
				}
			}
			vpcConfigs = append(vpcConfigs, vpc)
		}
		req.SetAws(externalEonSdkAPI.AwsRestoreAccountConnectivityConfig{VpcConfigs: vpcConfigs})
	}

	if data.Gcp != nil {
		networkConfigs := make([]externalEonSdkAPI.GcpNetworkConnectivityConfig, 0, len(data.Gcp.NetworkConfigs))
		for _, n := range data.Gcp.NetworkConfigs {
			network := externalEonSdkAPI.GcpNetworkConnectivityConfig{
				Network:            n.Network.ValueString(),
				IsSharedVpc:        n.IsSharedVpc.ValueBool(),
				NetworkHostProject: n.NetworkHostProject.ValueString(),
			}
			for _, s := range n.SubnetsPerRegion {
				network.SubnetsPerRegion = append(network.SubnetsPerRegion, externalEonSdkAPI.GcpSubnetPerRegion{
					Region:     s.Region.ValueString(),
					SubnetName: s.SubnetName.ValueString(),
				})
			}
			networkConfigs = append(networkConfigs, network)
		}
		req.SetGcp(externalEonSdkAPI.GcpRestoreAccountConnectivityConfig{NetworkConfigs: networkConfigs})
	}

	return req, diags
}

// stringListToModel converts an SDK string slice into a Terraform list, mapping a nil/empty
// slice to a null list so it matches an omitted optional attribute in configuration.
func stringListToModel(ctx context.Context, values []string) (types.List, diag.Diagnostics) {
	if len(values) == 0 {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

// connectivityConfigToModel populates the aws/gcp blocks of the model from the SDK config.
func connectivityConfigToModel(ctx context.Context, config *externalEonSdkAPI.RestoreAccountConnectivityConfig, data *RestoreAccountConnectivityConfigResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	providerCfg := config.GetConfig()
	data.Aws = nil
	data.Gcp = nil

	if providerCfg.HasAws() {
		awsCfg := providerCfg.GetAws()
		awsModel := &awsConnectivityConfigModel{}
		for _, v := range awsCfg.GetVpcConfigs() {
			vpcModel := awsVpcConnectivityConfigModel{
				Region: types.StringValue(v.Region),
				Vpc:    types.StringValue(v.Vpc),
			}
			for _, s := range v.SubnetsPerAvailabilityZone {
				vpcModel.SubnetsPerAvailabilityZone = append(vpcModel.SubnetsPerAvailabilityZone, subnetPerAvailabilityZoneModel{
					AvailabilityZone: types.StringValue(s.AvailabilityZone),
					SubnetId:         types.StringValue(s.SubnetId),
				})
			}
			if v.SecurityGroups != nil {
				restoreServer, d := stringListToModel(ctx, v.SecurityGroups.RestoreServer)
				diags.Append(d...)
				restoredRds, d := stringListToModel(ctx, v.SecurityGroups.RestoredRdsInstance)
				diags.Append(d...)
				vpcModel.SecurityGroups = &resourceTypeToSecurityGroupModel{
					RestoreServer:       restoreServer,
					RestoredRdsInstance: restoredRds,
				}
			}
			awsModel.VpcConfigs = append(awsModel.VpcConfigs, vpcModel)
		}
		data.Aws = awsModel
	}

	if providerCfg.HasGcp() {
		gcpCfg := providerCfg.GetGcp()
		gcpModel := &gcpConnectivityConfigModel{}
		for _, n := range gcpCfg.GetNetworkConfigs() {
			networkModel := gcpNetworkConnectivityConfigModel{
				Network:            types.StringValue(n.Network),
				IsSharedVpc:        types.BoolValue(n.IsSharedVpc),
				NetworkHostProject: types.StringValue(n.NetworkHostProject),
			}
			for _, s := range n.SubnetsPerRegion {
				networkModel.SubnetsPerRegion = append(networkModel.SubnetsPerRegion, gcpSubnetPerRegionModel{
					Region:     types.StringValue(s.Region),
					SubnetName: types.StringValue(s.SubnetName),
				})
			}
			gcpModel.NetworkConfigs = append(gcpModel.NetworkConfigs, networkModel)
		}
		data.Gcp = gcpModel
	}

	return diags
}
