package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// parseMapAttribute extracts a types.Map of string values into a map[string]string.
// Returns nil if the map is null or empty.
func parseMapAttribute(ctx context.Context, m types.Map) (map[string]string, error) {
	if m.IsNull() || len(m.Elements()) == 0 {
		return nil, nil
	}
	raw := make(map[string]types.String, len(m.Elements()))
	diags := m.ElementsAs(ctx, &raw, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse map attribute")
	}
	result := make(map[string]string, len(raw))
	for k, v := range raw {
		result[k] = v.ValueString()
	}
	return result, nil
}

var _ resource.Resource = &RestoreJobResource{}
var _ resource.ResourceWithImportState = &RestoreJobResource{}

func NewRestoreJobResource() resource.Resource {
	return &RestoreJobResource{}
}

type RestoreJobResource struct {
	client *client.EonClient
}

type RestoreJobResourceModel struct {
	Id               types.String `tfsdk:"id"`
	RestoreType      types.String `tfsdk:"restore_type"`
	SnapshotId       types.String `tfsdk:"snapshot_id"`
	ResourceId       types.String `tfsdk:"resource_id"`
	RestoreAccountId types.String `tfsdk:"restore_account_id"`

	// Restore type specific configuration blocks — AWS
	EbsConfig         *EbsRestoreConfig         `tfsdk:"ebs_config"`
	EbsSnapshotConfig *EbsSnapshotRestoreConfig `tfsdk:"ebs_snapshot_config"`
	Ec2Config         *Ec2RestoreConfig         `tfsdk:"ec2_config"`
	RdsConfig         *RdsRestoreConfig         `tfsdk:"rds_config"`
	S3BucketConfig    *S3BucketRestoreConfig    `tfsdk:"s3_bucket_config"`
	S3FileConfig      *S3FileRestoreConfig      `tfsdk:"s3_file_config"`
	DynamoDBConfig    *DynamoDBRestoreConfig    `tfsdk:"dynamodb_config"`

	// Restore type specific configuration blocks — Azure
	AzureDiskConfig *AzureDiskRestoreConfig `tfsdk:"azure_disk_config"`
	AzureVmConfig   *AzureVmRestoreConfig   `tfsdk:"azure_vm_config"`
	AzureSqlConfig  *AzureSqlRestoreConfig  `tfsdk:"azure_sql_config"`

	// Restore type specific configuration blocks — GCP
	GcpVmConfig              *GcpVmRestoreConfig              `tfsdk:"gcp_vm_config"`
	GcpDiskConfig            *GcpDiskRestoreConfig            `tfsdk:"gcp_disk_config"`
	GcpCloudSqlConfig        *GcpCloudSqlRestoreConfig        `tfsdk:"gcp_cloud_sql_config"`
	GcsBucketConfig          *GcsBucketRestoreConfig          `tfsdk:"gcs_bucket_config"`
	GcsFileConfig            *GcsFileRestoreConfig            `tfsdk:"gcs_file_config"`
	GcpBigQueryDatasetConfig *GcpBigQueryDatasetRestoreConfig `tfsdk:"gcp_bigquery_restore_dataset_config"`

	// Common fields
	TimeoutMinutes    types.Int64 `tfsdk:"timeout_minutes"`
	WaitForCompletion types.Bool  `tfsdk:"wait_for_completion"`

	// Job status fields (computed)
	JobId           types.String `tfsdk:"job_id"`
	Status          types.String `tfsdk:"status"`
	StatusMessage   types.String `tfsdk:"status_message"`
	CreatedAt       types.String `tfsdk:"created_at"`
	StartedAt       types.String `tfsdk:"started_at"`
	CompletedAt     types.String `tfsdk:"completed_at"`
	DurationSeconds types.Int64  `tfsdk:"duration_seconds"`
}

type EbsRestoreConfig struct {
	ProviderVolumeId           types.String `tfsdk:"provider_volume_id"`
	AvailabilityZone           types.String `tfsdk:"availability_zone"`
	VolumeType                 types.String `tfsdk:"volume_type"`
	VolumeSize                 types.Int64  `tfsdk:"volume_size"` // Size in bytes
	Iops                       types.Int64  `tfsdk:"iops"`
	Throughput                 types.Int64  `tfsdk:"throughput"`
	Description                types.String `tfsdk:"description"`
	VolumeEncryptionKeyId      types.String `tfsdk:"volume_encryption_key_id"`
	EnvironmentEncryptionKeyId types.String `tfsdk:"environment_encryption_key_id"`
	Tags                       types.Map    `tfsdk:"tags"`
}

type EbsSnapshotRestoreConfig struct {
	ProviderVolumeId        types.String `tfsdk:"provider_volume_id"`
	Region                  types.String `tfsdk:"region"`
	SnapshotEncryptionKeyId types.String `tfsdk:"snapshot_encryption_key_id"`
	Description             types.String `tfsdk:"description"`
	Tags                    types.Map    `tfsdk:"tags"`
}

type DynamoDBRestoreConfig struct {
	RestoreRegion      types.String `tfsdk:"restore_region"`
	RestoredName       types.String `tfsdk:"restored_name"`
	EncryptionKeyId    types.String `tfsdk:"encryption_key_id"`
	WriteCapacityUnits types.Int64  `tfsdk:"write_capacity_units"`
	Tags               types.Map    `tfsdk:"tags"`
}

type AzureDiskRestoreConfig struct {
	ProviderDiskId    types.String `tfsdk:"provider_disk_id"`
	Region            types.String `tfsdk:"region"`
	ResourceGroupName types.String `tfsdk:"resource_group_name"`
	Name              types.String `tfsdk:"name"`
	DiskType          types.String `tfsdk:"disk_type"`
	Tier              types.String `tfsdk:"tier"`
	HyperVGeneration  types.String `tfsdk:"hyper_v_generation"`
	SizeBytes         types.Int64  `tfsdk:"size_bytes"`
	Tags              types.Map    `tfsdk:"tags"`
}

type AzureVmRestoreConfig struct {
	Region                    types.String `tfsdk:"region"`
	ResourceGroupName         types.String `tfsdk:"resource_group_name"`
	VmName                    types.String `tfsdk:"vm_name"`
	VmSize                    types.String `tfsdk:"vm_size"`
	NetworkInterface          types.String `tfsdk:"network_interface"`
	StartInstanceAfterRestore types.Bool   `tfsdk:"start_instance_after_restore"`
	Tags                      types.Map    `tfsdk:"tags"`
	Disks                     types.List   `tfsdk:"disks"`
}

type AzureVmDiskRestoreParam struct {
	ProviderDiskId   types.String `tfsdk:"provider_disk_id"`
	Name             types.String `tfsdk:"name"`
	DiskType         types.String `tfsdk:"disk_type"`
	Tier             types.String `tfsdk:"tier"`
	HyperVGeneration types.String `tfsdk:"hyper_v_generation"`
	SizeBytes        types.Int64  `tfsdk:"size_bytes"`
	Tags             types.Map    `tfsdk:"tags"`
}

type AzureSqlRestoreConfig struct {
	Region            types.String `tfsdk:"region"`
	ResourceGroupName types.String `tfsdk:"resource_group_name"`
	ServerName        types.String `tfsdk:"server_name"`
	AdminUserName     types.String `tfsdk:"admin_user_name"`
	Tags              types.Map    `tfsdk:"tags"`
}

type Ec2RestoreConfig struct {
	Region              types.String `tfsdk:"region"`
	InstanceType        types.String `tfsdk:"instance_type"`
	SubnetId            types.String `tfsdk:"subnet_id"`
	SecurityGroupIds    types.List   `tfsdk:"security_group_ids"`
	Tags                types.Map    `tfsdk:"tags"`
	VolumeRestoreParams types.List   `tfsdk:"volume_restore_params"`
}

type RdsRestoreConfig struct {
	DbInstanceIdentifier  types.String `tfsdk:"db_instance_identifier"`
	DbInstanceClass       types.String `tfsdk:"db_instance_class"`
	Engine                types.String `tfsdk:"engine"`
	Region                types.String `tfsdk:"region"`
	SubnetGroupName       types.String `tfsdk:"subnet_group_name"`
	VpcSecurityGroupIds   types.List   `tfsdk:"vpc_security_group_ids"`
	AllocatedStorage      types.Int64  `tfsdk:"allocated_storage"`
	StorageType           types.String `tfsdk:"storage_type"`
	Tags                  types.Map    `tfsdk:"tags"`
	BackupRetentionPeriod types.Int64  `tfsdk:"backup_retention_period"`
	MultiAz               types.Bool   `tfsdk:"multi_az"`
	PubliclyAccessible    types.Bool   `tfsdk:"publicly_accessible"`
	StorageEncrypted      types.Bool   `tfsdk:"storage_encrypted"`
	KmsKeyId              types.String `tfsdk:"kms_key_id"`
}

type S3BucketRestoreConfig struct {
	BucketName types.String `tfsdk:"bucket_name"`
	KeyPrefix  types.String `tfsdk:"key_prefix"`
	KmsKeyId   types.String `tfsdk:"kms_key_id"`
}

type S3FileRestoreConfig struct {
	BucketName types.String `tfsdk:"bucket_name"`
	KeyPrefix  types.String `tfsdk:"key_prefix"`
	KmsKeyId   types.String `tfsdk:"kms_key_id"`
	Files      types.List   `tfsdk:"files"`
}

type VolumeRestoreParam struct {
	ProviderVolumeId types.String `tfsdk:"provider_volume_id"`
	VolumeType       types.String `tfsdk:"volume_type"`
	VolumeSize       types.Int64  `tfsdk:"volume_size"` // Size in bytes
	Iops             types.Int64  `tfsdk:"iops"`
	Throughput       types.Int64  `tfsdk:"throughput"`
	Description      types.String `tfsdk:"description"`
	KmsKeyId         types.String `tfsdk:"kms_key_id"`
}

type FileRestoreParam struct {
	Path        types.String `tfsdk:"path"`
	IsDirectory types.Bool   `tfsdk:"is_directory"`
}

// GCP restore config types

type GcpVmRestoreConfig struct {
	Zone                      types.String `tfsdk:"zone"`
	MachineType               types.String `tfsdk:"machine_type"`
	Name                      types.String `tfsdk:"name"`
	NetworkName               types.String `tfsdk:"network_name"`
	SubnetName                types.String `tfsdk:"subnet_name"`
	NetworkHostProject        types.String `tfsdk:"network_host_project"`
	Labels                    types.Map    `tfsdk:"labels"`
	StartInstanceAfterRestore types.Bool   `tfsdk:"start_instance_after_restore"`
	Disks                     types.List   `tfsdk:"disks"`
}

type GcpDiskRestoreParam struct {
	ProviderDiskId  types.String `tfsdk:"provider_disk_id"`
	Name            types.String `tfsdk:"name"`
	DiskType        types.String `tfsdk:"disk_type"`
	SizeBytes       types.Int64  `tfsdk:"size_bytes"`
	Iops            types.Int64  `tfsdk:"iops"`
	Throughput      types.Int64  `tfsdk:"throughput"`
	Description     types.String `tfsdk:"description"`
	Labels          types.Map    `tfsdk:"labels"`
	EncryptionKeyId types.String `tfsdk:"encryption_key_id"`
}

type GcpDiskRestoreConfig struct {
	ProviderDiskId  types.String `tfsdk:"provider_disk_id"`
	Zone            types.String `tfsdk:"zone"`
	Name            types.String `tfsdk:"name"`
	DiskType        types.String `tfsdk:"disk_type"`
	SizeBytes       types.Int64  `tfsdk:"size_bytes"`
	Iops            types.Int64  `tfsdk:"iops"`
	Throughput      types.Int64  `tfsdk:"throughput"`
	Description     types.String `tfsdk:"description"`
	Labels          types.Map    `tfsdk:"labels"`
	EncryptionKeyId types.String `tfsdk:"encryption_key_id"`
}

type GcpCloudSqlRestoreConfig struct {
	Zone               types.String `tfsdk:"zone"`
	Name               types.String `tfsdk:"name"`
	NetworkType        types.String `tfsdk:"network_type"`
	NetworkName        types.String `tfsdk:"network_name"`
	NetworkHostProject types.String `tfsdk:"network_host_project"`
	Labels             types.Map    `tfsdk:"labels"`
}

type GcsBucketRestoreConfig struct {
	BucketName types.String `tfsdk:"bucket_name"`
	KeyPrefix  types.String `tfsdk:"key_prefix"`
}

type GcsFileRestoreConfig struct {
	BucketName types.String `tfsdk:"bucket_name"`
	KeyPrefix  types.String `tfsdk:"key_prefix"`
	Files      types.List   `tfsdk:"files"`
}

// BigQuery restore config types

type GcpBigQueryDatasetRestoreConfig struct {
	DatasetId types.String `tfsdk:"dataset_id"`
	Location  types.String `tfsdk:"location"`
	Tables    types.List   `tfsdk:"tables"`
}

type GcpBigQueryTableParam struct {
	TableId types.String `tfsdk:"table_id"`
}

func (r *RestoreJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_job"
}

func (r *RestoreJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers a restore job to restore data from an Eon snapshot. This operation is asynchronous and returns a job ID that can be used to track the progress of the restore job.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Restore job ID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"restore_type": schema.StringAttribute{
				MarkdownDescription: "Type of restore job: `full` for full resource restore, `partial` for partial restore, `ebs_snapshot` for restore-to-native-EBS-snapshot.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"snapshot_id": schema.StringAttribute{
				MarkdownDescription: "ID of the Eon snapshot to restore from.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the resource to restore from (defaults to snapshot_id if not provided).",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"timeout_minutes": schema.Int64Attribute{
				MarkdownDescription: "Timeout in minutes for restore operation.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
			},
			"wait_for_completion": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for completion.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"job_id": schema.StringAttribute{
				MarkdownDescription: "Job ID.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Current status of the restore job. Possible values: `JOB_UNSPECIFIED`, `JOB_PENDING`, `JOB_RUNNING`, `JOB_COMPLETED`, `JOB_FAILED`, `JOB_PARTIAL`.",
				Computed:            true,
			},
			"status_message": schema.StringAttribute{
				MarkdownDescription: "Message that gives additional details about the job status, if applicable.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the job was created.",
				Computed:            true,
			},
			"started_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the job started.",
				Computed:            true,
			},
			"completed_at": schema.StringAttribute{
				MarkdownDescription: "Date and time the job finished.",
				Computed:            true,
			},
			"duration_seconds": schema.Int64Attribute{
				MarkdownDescription: "How long the job took, in seconds.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"ebs_config": schema.SingleNestedBlock{
				MarkdownDescription: "EBS volume restore configuration. Required when restoring AWS EC2 volume with `partial` restore type.",
				Attributes: map[string]schema.Attribute{
					"provider_volume_id": schema.StringAttribute{
						MarkdownDescription: "Cloud-provider-assigned ID of the volume to restore.",
						Optional:            true,
					},
					"availability_zone": schema.StringAttribute{
						MarkdownDescription: "Availability zone to restore the volume to.",
						Optional:            true,
					},
					"volume_type": schema.StringAttribute{
						MarkdownDescription: "EBS volume type (gp2, gp3, io1, io2, etc.).",
						Optional:            true,
					},
					"volume_size": schema.Int64Attribute{
						MarkdownDescription: "Volume size in bytes.",
						Optional:            true,
					},
					"iops": schema.Int64Attribute{
						MarkdownDescription: "IOPS for volume (required for io1/io2).",
						Optional:            true,
					},
					"throughput": schema.Int64Attribute{
						MarkdownDescription: "Throughput for gp3 volumes.",
						Optional:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description to apply to the restored volume.",
						Optional:            true,
					},
					"volume_encryption_key_id": schema.StringAttribute{
						MarkdownDescription: "ID of the KMS key you want Eon to use for encrypting the restored volume.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("alias/aws/ebs"),
					},
					"environment_encryption_key_id": schema.StringAttribute{
						MarkdownDescription: "KMS key ID for environment encryption.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags to apply to the restored volume as key-value pairs, where key and value are both strings.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
			"ebs_snapshot_config": schema.SingleNestedBlock{
				MarkdownDescription: "EBS snapshot restore configuration. Required when restoring an AWS volume to a native EBS snapshot (`restore_type` = `ebs_snapshot`).",
				Attributes: map[string]schema.Attribute{
					"provider_volume_id": schema.StringAttribute{
						MarkdownDescription: "Cloud-provider-assigned ID of the volume to convert to an EBS snapshot.",
						Optional:            true,
					},
					"region": schema.StringAttribute{
						MarkdownDescription: "Region to create the EBS snapshot in.",
						Optional:            true,
					},
					"snapshot_encryption_key_id": schema.StringAttribute{
						MarkdownDescription: "ID of the KMS key used to encrypt the EBS snapshot.",
						Optional:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description to apply to the EBS snapshot.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags to apply to the EBS snapshot as key-value pairs.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
			"dynamodb_config": schema.SingleNestedBlock{
				MarkdownDescription: "DynamoDB table restore configuration. Required when restoring an AWS DynamoDB table.",
				Attributes: map[string]schema.Attribute{
					"restore_region": schema.StringAttribute{
						MarkdownDescription: "Region to restore the DynamoDB table to.",
						Optional:            true,
					},
					"restored_name": schema.StringAttribute{
						MarkdownDescription: "Name to assign to the restored DynamoDB table.",
						Optional:            true,
					},
					"encryption_key_id": schema.StringAttribute{
						MarkdownDescription: "ID of the KMS key used to encrypt the restored table.",
						Optional:            true,
					},
					"write_capacity_units": schema.Int64Attribute{
						MarkdownDescription: "Provisioned write capacity units for the restored table.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags to apply to the restored table as key-value pairs.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
			"azure_disk_config": schema.SingleNestedBlock{
				MarkdownDescription: "Azure disk restore configuration. Required when restoring an Azure disk.",
				Attributes: map[string]schema.Attribute{
					"provider_disk_id": schema.StringAttribute{
						MarkdownDescription: "Cloud-provider-assigned ID of the disk to restore.",
						Optional:            true,
					},
					"region": schema.StringAttribute{
						MarkdownDescription: "Region to restore the disk to.",
						Optional:            true,
					},
					"resource_group_name": schema.StringAttribute{
						MarkdownDescription: "Name of the resource group to restore to.",
						Optional:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name for the restored disk.",
						Optional:            true,
					},
					"disk_type": schema.StringAttribute{
						MarkdownDescription: "Azure disk type (for example, Premium_LRS).",
						Optional:            true,
					},
					"tier": schema.StringAttribute{
						MarkdownDescription: "Azure disk tier.",
						Optional:            true,
					},
					"hyper_v_generation": schema.StringAttribute{
						MarkdownDescription: "Hyper-V generation of the disk (V1 or V2).",
						Optional:            true,
					},
					"size_bytes": schema.Int64Attribute{
						MarkdownDescription: "Size of the restored disk, in bytes.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags to apply to the restored disk as key-value pairs.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
			"azure_vm_config": schema.SingleNestedBlock{
				MarkdownDescription: "Azure VM instance restore configuration. Required when restoring an Azure Virtual Machine with `full` restore type.",
				Attributes: map[string]schema.Attribute{
					"region": schema.StringAttribute{
						MarkdownDescription: "Region to restore the VM to.",
						Optional:            true,
					},
					"resource_group_name": schema.StringAttribute{
						MarkdownDescription: "Name of the resource group to restore to.",
						Optional:            true,
					},
					"vm_name": schema.StringAttribute{
						MarkdownDescription: "Name for the restored VM.",
						Optional:            true,
					},
					"vm_size": schema.StringAttribute{
						MarkdownDescription: "Size of the restored VM (for example, Standard_D2s_v3).",
						Optional:            true,
					},
					"network_interface": schema.StringAttribute{
						MarkdownDescription: "Name of the network interface to use.",
						Optional:            true,
					},
					"start_instance_after_restore": schema.BoolAttribute{
						MarkdownDescription: "Whether to start the VM after restoring it.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags to apply to the restored VM as key-value pairs.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"disks": schema.ListNestedBlock{
						MarkdownDescription: "Disks to restore and attach to the restored VM.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"provider_disk_id": schema.StringAttribute{
									MarkdownDescription: "Cloud-provider-assigned ID of the disk to restore.",
									Optional:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "Name for the restored disk.",
									Optional:            true,
								},
								"disk_type": schema.StringAttribute{
									MarkdownDescription: "Azure disk type.",
									Optional:            true,
								},
								"tier": schema.StringAttribute{
									MarkdownDescription: "Azure disk tier.",
									Optional:            true,
								},
								"hyper_v_generation": schema.StringAttribute{
									MarkdownDescription: "Hyper-V generation of the disk (V1 or V2).",
									Optional:            true,
								},
								"size_bytes": schema.Int64Attribute{
									MarkdownDescription: "Size of the restored disk, in bytes.",
									Optional:            true,
								},
								"tags": schema.MapAttribute{
									MarkdownDescription: "Tags to apply to the restored disk.",
									ElementType:         types.StringType,
									Optional:            true,
								},
							},
						},
					},
				},
			},
			"azure_sql_config": schema.SingleNestedBlock{
				MarkdownDescription: "Azure SQL database restore configuration. Required when restoring an Azure SQL database.",
				Attributes: map[string]schema.Attribute{
					"region": schema.StringAttribute{
						MarkdownDescription: "Region to restore the database to.",
						Optional:            true,
					},
					"resource_group_name": schema.StringAttribute{
						MarkdownDescription: "Name of the resource group to restore to.",
						Optional:            true,
					},
					"server_name": schema.StringAttribute{
						MarkdownDescription: "Name of the Azure SQL server to restore to.",
						Optional:            true,
					},
					"admin_user_name": schema.StringAttribute{
						MarkdownDescription: "Administrator username for the restored database server.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags to apply to the restored database as key-value pairs.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
			"ec2_config": schema.SingleNestedBlock{
				MarkdownDescription: "EC2 instance restore configuration. Required when restoring AWS EC2 instance with `full` restore type.",
				Attributes: map[string]schema.Attribute{
					"region": schema.StringAttribute{
						MarkdownDescription: "Region to restore the instance to.",
						Optional:            true,
					},
					"instance_type": schema.StringAttribute{
						MarkdownDescription: "Instance type to use for the restored instance.",
						Optional:            true,
					},
					"subnet_id": schema.StringAttribute{
						MarkdownDescription: "Subnet ID to associate with the restored instance.",
						Optional:            true,
					},
					"security_group_ids": schema.ListAttribute{
						MarkdownDescription: "List of security group IDs to associate with the restored instance.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags to apply to the restored instance as key-value pairs, where key and value are both strings.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"volume_restore_params": schema.ListNestedBlock{
						MarkdownDescription: "Volumes to restore and attach to the restored instance. Each item corresponds to a volume to be restored, where `provider_volume_id` matches the volume's ID at the time of the snapshot. The root volume must be present in the list.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"provider_volume_id": schema.StringAttribute{
									MarkdownDescription: "Cloud-provider-assigned ID of the volume to restore.",
									Optional:            true,
								},
								"volume_type": schema.StringAttribute{
									MarkdownDescription: "EBS volume type (gp2, gp3, io1, io2, etc.).",
									Optional:            true,
								},
								"volume_size": schema.Int64Attribute{
									MarkdownDescription: "Volume size in bytes.",
									Optional:            true,
								},
								"iops": schema.Int64Attribute{
									MarkdownDescription: "IOPS for volume (required for io1/io2).",
									Optional:            true,
								},
								"throughput": schema.Int64Attribute{
									MarkdownDescription: "Throughput for gp3 volumes.",
									Optional:            true,
								},
								"description": schema.StringAttribute{
									MarkdownDescription: "Optional description for the restored volume.",
									Optional:            true,
								},
								"kms_key_id": schema.StringAttribute{
									MarkdownDescription: "ARN of the KMS key for encrypting the restored volume.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
			"rds_config": schema.SingleNestedBlock{
				MarkdownDescription: "RDS database restore configuration. Required when restoring AWS RDS database.",
				Attributes: map[string]schema.Attribute{
					"db_instance_identifier": schema.StringAttribute{
						MarkdownDescription: "Name to assign to the restored resource.",
						Optional:            true,
					},
					"db_instance_class": schema.StringAttribute{
						MarkdownDescription: "DB instance class (for example, db.t3.micro).",
						Optional:            true,
					},
					"engine": schema.StringAttribute{
						MarkdownDescription: "Database engine (for example, mysql, postgres).",
						Optional:            true,
					},
					"region": schema.StringAttribute{
						MarkdownDescription: "Region to restore to.",
						Optional:            true,
					},
					"subnet_group_name": schema.StringAttribute{
						MarkdownDescription: "Subnet group ID to associate with the restored resource. Must be in the same VPC of `vpc_security_group_ids`.",
						Optional:            true,
					},
					"vpc_security_group_ids": schema.ListAttribute{
						MarkdownDescription: "List of security group IDs to associate with the restored resource. Must be in the same VPC of `subnet_group_name`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"allocated_storage": schema.Int64Attribute{
						MarkdownDescription: "Allocated storage in GiB.",
						Optional:            true,
					},
					"storage_type": schema.StringAttribute{
						MarkdownDescription: "Storage type (gp2, gp3, io1, etc.).",
						Optional:            true,
					},
					"backup_retention_period": schema.Int64Attribute{
						MarkdownDescription: "Backup retention period in days.",
						Optional:            true,
					},
					"multi_az": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable Multi-AZ deployment.",
						Optional:            true,
					},
					"publicly_accessible": schema.BoolAttribute{
						MarkdownDescription: "Whether the database is publicly accessible.",
						Optional:            true,
					},
					"storage_encrypted": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable storage encryption.",
						Optional:            true,
					},
					"kms_key_id": schema.StringAttribute{
						MarkdownDescription: "ID of the key you want Eon to use for encrypting the restored resource.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags to apply to the restored instance as key-value pairs, where key and value are both strings.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
			"s3_bucket_config": schema.SingleNestedBlock{
				MarkdownDescription: "S3 bucket restore configuration. Required when restoring AWS S3 bucket with `full` restore type.",
				Attributes: map[string]schema.Attribute{
					"bucket_name": schema.StringAttribute{
						MarkdownDescription: "Name of an existing bucket to restore the data to.",
						Optional:            true,
					},
					"key_prefix": schema.StringAttribute{
						MarkdownDescription: "Prefix to add to the restore path. If you don't specify a prefix, the files are restored to their respective folders in the original file tree, starting from the root of the bucket.",
						Optional:            true,
					},
					"kms_key_id": schema.StringAttribute{
						MarkdownDescription: "ID of the key you want Eon to use for encrypting the restored files.",
						Optional:            true,
					},
				},
			},
			"s3_file_config": schema.SingleNestedBlock{
				MarkdownDescription: "S3 file restore configuration. Required when restoring AWS S3 files with partial restore type.",
				Attributes: map[string]schema.Attribute{
					"bucket_name": schema.StringAttribute{
						MarkdownDescription: "Name of an existing bucket to restore the files to.",
						Optional:            true,
					},
					"key_prefix": schema.StringAttribute{
						MarkdownDescription: "Prefix to add to the restore path. If you don't specify a prefix, the files are restored to their respective folders in the original file tree, starting from the root of the bucket.",
						Optional:            true,
					},
					"kms_key_id": schema.StringAttribute{
						MarkdownDescription: "ID of the key you want Eon to use for encrypting the restored files.",
						Optional:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"files": schema.ListNestedBlock{
						MarkdownDescription: "List of file paths to restore.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"path": schema.StringAttribute{
									MarkdownDescription: "Absolute path to the file or directory to restore.",
									Optional:            true,
								},
								"is_directory": schema.BoolAttribute{
									MarkdownDescription: "Whether `path` is a directory. If `true`, Eon restores all files in all subdirectories under the path. If `false`, Eon restores only the file at the path.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
			// GCP restore configuration blocks
			"gcp_vm_config": schema.SingleNestedBlock{
				MarkdownDescription: "GCP VM instance restore configuration. Required when restoring a GCP Compute Engine instance with `full` restore type.",
				Attributes: map[string]schema.Attribute{
					"zone": schema.StringAttribute{
						MarkdownDescription: "Zone to restore the VM instance to (e.g. `us-central1-a`).",
						Optional:            true,
					},
					"machine_type": schema.StringAttribute{
						MarkdownDescription: "Machine type to use for the restored instance (e.g. `e2-medium`).",
						Optional:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name for the restored VM instance.",
						Optional:            true,
					},
					"network_name": schema.StringAttribute{
						MarkdownDescription: "Name of the VPC network to use.",
						Optional:            true,
					},
					"subnet_name": schema.StringAttribute{
						MarkdownDescription: "Name of the subnet to use.",
						Optional:            true,
					},
					"network_host_project": schema.StringAttribute{
						MarkdownDescription: "ID of the project that hosts the VPC network. Applicable only when restoring to a shared VPC network.",
						Optional:            true,
					},
					"labels": schema.MapAttribute{
						MarkdownDescription: "Labels to apply to the restored VM as key-value pairs. The label `\"eon-restore\": \"true\"` is always applied automatically.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"start_instance_after_restore": schema.BoolAttribute{
						MarkdownDescription: "Whether to start the VM instance after restoring it. If `false`, the VM will be created in a stopped state. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
				},
				Blocks: map[string]schema.Block{
					"disks": schema.ListNestedBlock{
						MarkdownDescription: "Disks to restore and attach to the restored instance. Each item corresponds to a disk, where `provider_disk_id` matches the disk's ID at the time of the snapshot. The boot disk must be in the list.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"provider_disk_id": schema.StringAttribute{
									MarkdownDescription: "Cloud-provider-assigned ID of the disk to restore.",
									Optional:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "Disk name.",
									Optional:            true,
								},
								"disk_type": schema.StringAttribute{
									MarkdownDescription: "Disk type (e.g. `pd-standard`, `pd-ssd`, `pd-balanced`, `pd-extreme`).",
									Optional:            true,
								},
								"size_bytes": schema.Int64Attribute{
									MarkdownDescription: "Size of the disk in bytes.",
									Optional:            true,
								},
								"iops": schema.Int64Attribute{
									MarkdownDescription: "Provisioned IOPS for the disk. Applicable only when `disk_type` is `pd-extreme`.",
									Optional:            true,
								},
								"throughput": schema.Int64Attribute{
									MarkdownDescription: "Disk throughput. Defaults to the original throughput captured by the snapshot.",
									Optional:            true,
								},
								"description": schema.StringAttribute{
									MarkdownDescription: "Description for the restored disk.",
									Optional:            true,
								},
								"labels": schema.MapAttribute{
									MarkdownDescription: "Labels to apply to the restored disk as key-value pairs.",
									ElementType:         types.StringType,
									Optional:            true,
								},
								"encryption_key_id": schema.StringAttribute{
									MarkdownDescription: "ID of the customer-managed encryption key (CMEK) to use for the disk.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
			"gcp_disk_config": schema.SingleNestedBlock{
				MarkdownDescription: "GCP disk restore configuration. Required when restoring a GCP Compute Engine disk with `partial` restore type, or a standalone GCP disk.",
				Attributes: map[string]schema.Attribute{
					"provider_disk_id": schema.StringAttribute{
						MarkdownDescription: "Cloud-provider-assigned ID of the disk to restore.",
						Optional:            true,
					},
					"zone": schema.StringAttribute{
						MarkdownDescription: "Zone to restore the disk to (e.g. `us-central1-a`).",
						Optional:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name for the restored disk.",
						Optional:            true,
					},
					"disk_type": schema.StringAttribute{
						MarkdownDescription: "Disk type (e.g. `pd-standard`, `pd-ssd`, `pd-balanced`, `pd-extreme`).",
						Optional:            true,
					},
					"size_bytes": schema.Int64Attribute{
						MarkdownDescription: "Size of the disk in bytes.",
						Optional:            true,
					},
					"iops": schema.Int64Attribute{
						MarkdownDescription: "Provisioned IOPS for the disk. Applicable only when `disk_type` is `pd-extreme`.",
						Optional:            true,
					},
					"throughput": schema.Int64Attribute{
						MarkdownDescription: "Disk throughput. Defaults to the original throughput captured by the snapshot.",
						Optional:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description for the restored disk.",
						Optional:            true,
					},
					"labels": schema.MapAttribute{
						MarkdownDescription: "Labels to apply to the restored disk as key-value pairs.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"encryption_key_id": schema.StringAttribute{
						MarkdownDescription: "ID of the customer-managed encryption key (CMEK) to use for the disk.",
						Optional:            true,
					},
				},
			},
			"gcp_cloud_sql_config": schema.SingleNestedBlock{
				MarkdownDescription: "GCP Cloud SQL restore configuration. Required when restoring a GCP Cloud SQL instance.",
				Attributes: map[string]schema.Attribute{
					"zone": schema.StringAttribute{
						MarkdownDescription: "Zone to restore the Cloud SQL instance to (e.g. `us-central1-a`).",
						Optional:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name for the restored Cloud SQL instance.",
						Optional:            true,
					},
					"network_type": schema.StringAttribute{
						MarkdownDescription: "Network type for the Cloud SQL instance. Possible values: `PUBLIC`, `PRIVATE`.",
						Optional:            true,
					},
					"network_name": schema.StringAttribute{
						MarkdownDescription: "Name of the VPC network to use. Required when `network_type` is `PRIVATE`.",
						Optional:            true,
					},
					"network_host_project": schema.StringAttribute{
						MarkdownDescription: "ID of the project that hosts the VPC network. Applicable only when restoring to a shared VPC network.",
						Optional:            true,
					},
					"labels": schema.MapAttribute{
						MarkdownDescription: "Labels to apply to the restored Cloud SQL instance as key-value pairs. The label `\"eon-restore\": \"true\"` is always applied automatically.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
			"gcs_bucket_config": schema.SingleNestedBlock{
				MarkdownDescription: "GCS bucket restore configuration. Required when restoring a GCP Cloud Storage bucket with `full` restore type.",
				Attributes: map[string]schema.Attribute{
					"bucket_name": schema.StringAttribute{
						MarkdownDescription: "Name of an existing GCS bucket to restore the data to.",
						Optional:            true,
					},
					"key_prefix": schema.StringAttribute{
						MarkdownDescription: "Prefix to add to the restore path. If you don't specify a prefix, the files are restored to their respective folders in the original file tree, starting from the root of the bucket.",
						Optional:            true,
					},
				},
			},
			"gcs_file_config": schema.SingleNestedBlock{
				MarkdownDescription: "GCS file restore configuration. Required when restoring GCP Cloud Storage files with `partial` restore type.",
				Attributes: map[string]schema.Attribute{
					"bucket_name": schema.StringAttribute{
						MarkdownDescription: "Name of an existing GCS bucket to restore the files to.",
						Optional:            true,
					},
					"key_prefix": schema.StringAttribute{
						MarkdownDescription: "Prefix to add to the restore path. If you don't specify a prefix, the files are restored to their respective folders in the original file tree, starting from the root of the bucket.",
						Optional:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"files": schema.ListNestedBlock{
						MarkdownDescription: "List of file paths to restore.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"path": schema.StringAttribute{
									MarkdownDescription: "Absolute path to the file or directory to restore.",
									Optional:            true,
								},
								"is_directory": schema.BoolAttribute{
									MarkdownDescription: "Whether `path` is a directory. If `true`, Eon restores all files in all subdirectories under the path. If `false`, Eon restores only the file at the path.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
			// BigQuery restore configuration blocks
			"gcp_bigquery_restore_dataset_config": schema.SingleNestedBlock{
				MarkdownDescription: "GCP BigQuery dataset restore configuration. Required when restoring a BigQuery dataset. Provide a target `dataset_id` and `location`. When no table filter is specified, all tables in the dataset are restored. When tables are specified, only matching tables are restored.",
				Attributes: map[string]schema.Attribute{
					"dataset_id": schema.StringAttribute{
						MarkdownDescription: "Target BigQuery dataset ID for the restore (e.g. `my_dataset_restored`).",
						Optional:            true,
					},
					"location": schema.StringAttribute{
						MarkdownDescription: "GCP location for the restored dataset (e.g. `US`, `EU`, `us-central1`).",
						Optional:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"tables": schema.ListNestedBlock{
						MarkdownDescription: "Optional list of tables to restore. When omitted, all tables in the dataset are restored.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"table_id": schema.StringAttribute{
									MarkdownDescription: "BigQuery table ID to include in the restore.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *RestoreJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.EonClient, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *RestoreJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RestoreJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshot, err := r.client.GetSnapshot(ctx, data.SnapshotId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve snapshot with ID %s: %s", data.SnapshotId.ValueString(), err))
		return
	}

	// Set resource_id from snapshot
	resourceId := snapshot.GetResourceId()
	data.ResourceId = types.StringValue(resourceId)

	inventoryResource, err := r.client.GetResourceById(ctx, resourceId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to retrieve resource with ID %s: %s", resourceId, err))
		return
	}

	restoreType := data.RestoreType.ValueString()
	var jobId string

	// BigQuery accepts any restore_type — handle it before validating restore_type
	if inventoryResource.GetResourceType() == externalEonSdkAPI.GCP_BIG_QUERY {
		if data.GcpBigQueryDatasetConfig == nil {
			resp.Diagnostics.AddError("Configuration Error", "gcp_bigquery_restore_dataset_config is required when restoring GCP BigQuery datasets")
			return
		}
		jobId, err = r.createGcpBigQueryDatasetRestore(ctx, data, resourceId)
	} else {
		// Validate restore_type for all non-BigQuery resource types
		if restoreType != "full" && restoreType != "partial" && restoreType != "ebs_snapshot" {
			resp.Diagnostics.AddError("Configuration Error", fmt.Sprintf("Invalid restore_type: %s. Supported types: full, partial, ebs_snapshot", restoreType))
			return
		}

		// Route to the correct restore method based on resource type
		switch inventoryResource.GetResourceType() {
		// AWS resource types
		case externalEonSdkAPI.AWS_EC2, externalEonSdkAPI.AWS_EBS_VOLUME:
			if restoreType == "ebs_snapshot" || data.EbsSnapshotConfig != nil {
				if data.EbsSnapshotConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "ebs_snapshot_config is required when restore_type is 'ebs_snapshot'")
					return
				}
				jobId, err = r.createEbsSnapshotRestore(ctx, data, resourceId)
			} else if restoreType == "partial" {
				if data.EbsConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "ebs_config is required when restoring AWS EC2 volumes with restore_type 'partial'")
					return
				}
				jobId, err = r.createEbsVolumeRestore(ctx, data, resourceId)
			} else {
				if data.Ec2Config == nil {
					resp.Diagnostics.AddError("Configuration Error", "ec2_config is required when restoring AWS EC2 instances with restore_type 'full'")
					return
				}
				jobId, err = r.createEc2InstanceRestore(ctx, data, resourceId)
			}
		case externalEonSdkAPI.AWS_RDS:
			if data.RdsConfig == nil {
				resp.Diagnostics.AddError("Configuration Error", "rds_config is required when restoring AWS RDS databases")
				return
			}
			jobId, err = r.createRdsRestore(ctx, data, resourceId)
		case externalEonSdkAPI.AWS_S3:
			if restoreType == "full" {
				if data.S3BucketConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "s3_bucket_config is required when restoring AWS S3 buckets with restore_type 'full'")
					return
				}
				jobId, err = r.createS3BucketRestore(ctx, data, resourceId)
			} else {
				if data.S3FileConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "s3_file_config is required when restoring AWS S3 files with restore_type 'partial'")
					return
				}
				jobId, err = r.createS3FileRestore(ctx, data, resourceId)
			}
		case externalEonSdkAPI.AWS_DYNAMO_DB:
			if data.DynamoDBConfig == nil {
				resp.Diagnostics.AddError("Configuration Error", "dynamodb_config is required when restoring AWS DynamoDB tables")
				return
			}
			jobId, err = r.createDynamoDBTableRestore(ctx, data, resourceId)
		case externalEonSdkAPI.AZURE_DISK:
			if data.AzureDiskConfig == nil {
				resp.Diagnostics.AddError("Configuration Error", "azure_disk_config is required when restoring Azure disks")
				return
			}
			jobId, err = r.createAzureDiskRestore(ctx, data, resourceId)
		case externalEonSdkAPI.AZURE_VIRTUAL_MACHINE:
			if restoreType == "partial" {
				if data.AzureDiskConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "azure_disk_config is required when restoring Azure VM disks with restore_type 'partial'")
					return
				}
				jobId, err = r.createAzureDiskRestore(ctx, data, resourceId)
			} else {
				if data.AzureVmConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "azure_vm_config is required when restoring Azure VMs with restore_type 'full'")
					return
				}
				jobId, err = r.createAzureVmInstanceRestore(ctx, data, resourceId)
			}
		case externalEonSdkAPI.AZURE_SQL_DATABASE:
			if data.AzureSqlConfig == nil {
				resp.Diagnostics.AddError("Configuration Error", "azure_sql_config is required when restoring Azure SQL databases")
				return
			}
			jobId, err = r.createAzureSqlDatabaseRestore(ctx, data, resourceId)
		// GCP resource types
		case externalEonSdkAPI.GCP_COMPUTE_ENGINE_INSTANCE:
			if restoreType == "partial" {
				if data.GcpDiskConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "gcp_disk_config is required when restoring GCP Compute Engine disks with restore_type 'partial'")
					return
				}
				jobId, err = r.createGcpDiskRestore(ctx, data, resourceId)
			} else {
				if data.GcpVmConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "gcp_vm_config is required when restoring GCP Compute Engine instances with restore_type 'full'")
					return
				}
				jobId, err = r.createGcpVmInstanceRestore(ctx, data, resourceId)
			}
		case externalEonSdkAPI.GCP_DISK:
			if data.GcpDiskConfig == nil {
				resp.Diagnostics.AddError("Configuration Error", "gcp_disk_config is required when restoring GCP disks")
				return
			}
			jobId, err = r.createGcpDiskRestore(ctx, data, resourceId)
		case externalEonSdkAPI.GCP_CLOUD_SQL_INSTANCE:
			if data.GcpCloudSqlConfig == nil {
				resp.Diagnostics.AddError("Configuration Error", "gcp_cloud_sql_config is required when restoring GCP Cloud SQL instances")
				return
			}
			jobId, err = r.createGcpCloudSqlRestore(ctx, data, resourceId)
		case externalEonSdkAPI.GCP_CLOUD_STORAGE_BUCKET:
			if restoreType == "full" {
				if data.GcsBucketConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "gcs_bucket_config is required when restoring GCP Cloud Storage buckets with restore_type 'full'")
					return
				}
				jobId, err = r.createGcsBucketRestore(ctx, data, resourceId)
			} else {
				if data.GcsFileConfig == nil {
					resp.Diagnostics.AddError("Configuration Error", "gcs_file_config is required when restoring GCP Cloud Storage files with restore_type 'partial'")
					return
				}
				jobId, err = r.createGcsFileRestore(ctx, data, resourceId)
			}
		default:
			resp.Diagnostics.AddError("Configuration Error", fmt.Sprintf("Unsupported resource type: %s. Supported types: AWS_EC2, AWS_EBS_VOLUME, AWS_RDS, AWS_S3, AWS_DYNAMO_DB, AZURE_DISK, AZURE_VIRTUAL_MACHINE, AZURE_SQL_DATABASE, GCP_COMPUTE_ENGINE_INSTANCE, GCP_DISK, GCP_CLOUD_SQL_INSTANCE, GCP_CLOUD_STORAGE_BUCKET, GCP_BIG_QUERY", inventoryResource.GetResourceType()))
			return
		}
	} // end else (non-BigQuery)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to start restore job: %s", err))
		return
	}

	data.JobId = types.StringValue(jobId)
	data.Id = types.StringValue(jobId)
	data.Status = types.StringValue("JOB_PENDING")
	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Debug(ctx, "Started restore job", map[string]interface{}{
		"job_id":       jobId,
		"restore_type": restoreType,
		"snapshot_id":  data.SnapshotId.ValueString(),
	})

	// Initialize all computed fields to avoid "unknown" values
	data.StatusMessage = types.StringNull()
	data.StartedAt = types.StringNull()
	data.CompletedAt = types.StringNull()
	data.DurationSeconds = types.Int64Null()

	// Wait for completion if requested
	if data.WaitForCompletion.ValueBool() {
		timeout := time.Duration(data.TimeoutMinutes.ValueInt64()) * time.Minute
		finalJob, err := r.client.WaitForRestoreJobCompletion(ctx, jobId, timeout)
		if err != nil {
			tflog.Warn(ctx, "Restore job may still be running", map[string]interface{}{"error": err.Error()})
			data.StatusMessage = types.StringValue(err.Error())
			data.Status = types.StringValue("JOB_FAILED")

			// Try to get the actual job status to fill in details
			if actualJob, getErr := r.client.GetRestoreJob(ctx, jobId); getErr == nil {
				r.updateJobStatus(ctx, &data, actualJob)
			}
		} else {
			r.updateJobStatus(ctx, &data, finalJob)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreJobResource) createEbsVolumeRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.EbsConfig

	// Validate required fields for EBS volume restore
	if config.ProviderVolumeId.IsNull() || config.ProviderVolumeId.ValueString() == "" {
		return "", fmt.Errorf("provider_volume_id is required for EBS volume restore")
	}
	if config.AvailabilityZone.IsNull() || config.AvailabilityZone.ValueString() == "" {
		return "", fmt.Errorf("availability_zone is required for EBS volume restore")
	}
	if config.VolumeType.IsNull() || config.VolumeType.ValueString() == "" {
		return "", fmt.Errorf("volume_type is required for EBS volume restore")
	}
	if config.VolumeSize.IsNull() || config.VolumeSize.ValueInt64() == 0 {
		return "", fmt.Errorf("volume_size is required for EBS volume restore")
	}

	tags, err := parseMapAttribute(ctx, config.Tags)
	if err != nil {
		return "", err
	}

	// Build volume settings
	volumeSettings := externalEonSdkAPI.VolumeSettings{
		Type:      config.VolumeType.ValueString(),
		SizeBytes: config.VolumeSize.ValueInt64(),
	}

	if !config.Iops.IsNull() {
		i32, err := SafeInt32Conversion(config.Iops.ValueInt64())
		if err != nil {
			return "", err
		}
		volumeSettings.Iops = &i32
	}
	if !config.Throughput.IsNull() {
		t32, err := SafeInt32Conversion(config.Throughput.ValueInt64())
		if err != nil {
			return "", err
		}
		volumeSettings.Throughput = &t32
	}

	ebsTarget := &externalEonSdkAPI.EbsTarget{
		AvailabilityZone:      config.AvailabilityZone.ValueString(),
		VolumeEncryptionKeyId: config.VolumeEncryptionKeyId.ValueString(),
		VolumeSettings:        volumeSettings,
	}

	if !config.Description.IsNull() {
		desc := config.Description.ValueString()
		ebsTarget.Description = &desc
	}

	if tags != nil {
		ebsTarget.Tags = &tags
	}

	apiReq := externalEonSdkAPI.RestoreVolumeToEbsRequest{
		ProviderVolumeId: config.ProviderVolumeId.ValueString(),
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.EbsRestoreDestination{
			AwsEbs: ebsTarget,
		},
	}

	return r.client.StartVolumeRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createEbsSnapshotRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.EbsSnapshotConfig

	if config.ProviderVolumeId.IsNull() || config.ProviderVolumeId.ValueString() == "" {
		return "", fmt.Errorf("provider_volume_id is required for EBS snapshot restore")
	}
	if config.Region.IsNull() || config.Region.ValueString() == "" {
		return "", fmt.Errorf("region is required for EBS snapshot restore")
	}
	if config.SnapshotEncryptionKeyId.IsNull() || config.SnapshotEncryptionKeyId.ValueString() == "" {
		return "", fmt.Errorf("snapshot_encryption_key_id is required for EBS snapshot restore")
	}

	tags, err := parseMapAttribute(ctx, config.Tags)
	if err != nil {
		return "", err
	}

	target := &externalEonSdkAPI.EbsSnapshotTarget{
		Region:                  config.Region.ValueString(),
		SnapshotEncryptionKeyId: config.SnapshotEncryptionKeyId.ValueString(),
	}
	if !config.Description.IsNull() && config.Description.ValueString() != "" {
		desc := config.Description.ValueString()
		target.Description = &desc
	}
	if tags != nil {
		target.Tags = &tags
	}

	apiReq := externalEonSdkAPI.RestoreVolumeToEbsSnapshotRequest{
		ProviderVolumeId: config.ProviderVolumeId.ValueString(),
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.EbsSnapshotRestoreDestination{
			AwsEbs: target,
		},
	}

	return r.client.StartEbsSnapshotRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createDynamoDBTableRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.DynamoDBConfig

	if config.RestoreRegion.IsNull() || config.RestoreRegion.ValueString() == "" {
		return "", fmt.Errorf("restore_region is required for DynamoDB table restore")
	}
	if config.RestoredName.IsNull() || config.RestoredName.ValueString() == "" {
		return "", fmt.Errorf("restored_name is required for DynamoDB table restore")
	}

	tags, err := parseMapAttribute(ctx, config.Tags)
	if err != nil {
		return "", err
	}

	dest := &externalEonSdkAPI.AwsDynamoDBDestination{
		RestoreRegion: config.RestoreRegion.ValueString(),
		RestoredName:  config.RestoredName.ValueString(),
	}
	if !config.EncryptionKeyId.IsNull() && config.EncryptionKeyId.ValueString() != "" {
		key := config.EncryptionKeyId.ValueString()
		dest.EncryptionKeyId = &key
	}
	if !config.WriteCapacityUnits.IsNull() {
		wcu, err := SafeInt32Conversion(config.WriteCapacityUnits.ValueInt64())
		if err != nil {
			return "", err
		}
		dest.WriteCapacityUnits = &wcu
	}
	if tags != nil {
		dest.Tags = &tags
	}

	apiReq := externalEonSdkAPI.RestoreDynamoDBTableRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.DynamodbTableRestoreDestination{
			AwsDynamodb: dest,
		},
	}

	return r.client.StartDynamoDBTableRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func azureDiskSettingsFromModel(ctx context.Context, name, diskType, tier, hyperV types.String, sizeBytes types.Int64, tags types.Map) (externalEonSdkAPI.AzureDiskSettings, error) {
	settings := externalEonSdkAPI.AzureDiskSettings{
		Name: name.ValueString(),
		Type: diskType.ValueString(),
		Tier: tier.ValueString(),
	}
	if !hyperV.IsNull() && hyperV.ValueString() != "" {
		v := hyperV.ValueString()
		settings.HyperVGeneration = &v
	}
	if !sizeBytes.IsNull() && sizeBytes.ValueInt64() != 0 {
		v := sizeBytes.ValueInt64()
		settings.SizeBytes = &v
	}
	parsedTags, err := parseMapAttribute(ctx, tags)
	if err != nil {
		return settings, err
	}
	if parsedTags != nil {
		settings.Tags = &parsedTags
	}
	return settings, nil
}

func (r *RestoreJobResource) createAzureDiskRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.AzureDiskConfig

	if config.ProviderDiskId.IsNull() || config.ProviderDiskId.ValueString() == "" {
		return "", fmt.Errorf("provider_disk_id is required for Azure disk restore")
	}
	if config.Region.IsNull() || config.Region.ValueString() == "" {
		return "", fmt.Errorf("region is required for Azure disk restore")
	}
	if config.ResourceGroupName.IsNull() || config.ResourceGroupName.ValueString() == "" {
		return "", fmt.Errorf("resource_group_name is required for Azure disk restore")
	}
	if config.Name.IsNull() || config.Name.ValueString() == "" {
		return "", fmt.Errorf("name is required for Azure disk restore")
	}
	if config.DiskType.IsNull() || config.DiskType.ValueString() == "" {
		return "", fmt.Errorf("disk_type is required for Azure disk restore")
	}
	if config.Tier.IsNull() || config.Tier.ValueString() == "" {
		return "", fmt.Errorf("tier is required for Azure disk restore")
	}

	settings, err := azureDiskSettingsFromModel(ctx, config.Name, config.DiskType, config.Tier, config.HyperVGeneration, config.SizeBytes, config.Tags)
	if err != nil {
		return "", err
	}

	target := &externalEonSdkAPI.AzureDiskTarget{
		Region:            config.Region.ValueString(),
		ResourceGroupName: config.ResourceGroupName.ValueString(),
		Settings:          settings,
	}

	apiReq := externalEonSdkAPI.RestoreAzureDiskRequest{
		ProviderDiskId:   config.ProviderDiskId.ValueString(),
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.AzureDiskRestoreDestination{
			AzureDisk: target,
		},
	}

	return r.client.StartAzureDiskRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createAzureVmInstanceRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.AzureVmConfig

	if config.Region.IsNull() || config.Region.ValueString() == "" {
		return "", fmt.Errorf("region is required for Azure VM restore")
	}
	if config.ResourceGroupName.IsNull() || config.ResourceGroupName.ValueString() == "" {
		return "", fmt.Errorf("resource_group_name is required for Azure VM restore")
	}
	if config.VmName.IsNull() || config.VmName.ValueString() == "" {
		return "", fmt.Errorf("vm_name is required for Azure VM restore")
	}
	if config.VmSize.IsNull() || config.VmSize.ValueString() == "" {
		return "", fmt.Errorf("vm_size is required for Azure VM restore")
	}
	if config.Disks.IsNull() || len(config.Disks.Elements()) == 0 {
		return "", fmt.Errorf("disks is required for Azure VM restore")
	}

	tags, err := parseMapAttribute(ctx, config.Tags)
	if err != nil {
		return "", err
	}

	var diskParams []AzureVmDiskRestoreParam
	diags := config.Disks.ElementsAs(ctx, &diskParams, false)
	if diags.HasError() {
		return "", fmt.Errorf("failed to parse Azure VM disks")
	}

	disks := make([]externalEonSdkAPI.RestoreAzureInstanceDiskInput, 0, len(diskParams))
	for _, disk := range diskParams {
		if disk.ProviderDiskId.IsNull() || disk.ProviderDiskId.ValueString() == "" {
			return "", fmt.Errorf("provider_disk_id is required for each Azure VM disk")
		}
		if disk.Name.IsNull() || disk.Name.ValueString() == "" {
			return "", fmt.Errorf("name is required for each Azure VM disk")
		}
		if disk.DiskType.IsNull() || disk.DiskType.ValueString() == "" {
			return "", fmt.Errorf("disk_type is required for each Azure VM disk")
		}
		if disk.Tier.IsNull() || disk.Tier.ValueString() == "" {
			return "", fmt.Errorf("tier is required for each Azure VM disk")
		}
		settings, err := azureDiskSettingsFromModel(ctx, disk.Name, disk.DiskType, disk.Tier, disk.HyperVGeneration, disk.SizeBytes, disk.Tags)
		if err != nil {
			return "", err
		}
		disks = append(disks, externalEonSdkAPI.RestoreAzureInstanceDiskInput{
			ProviderDiskId: disk.ProviderDiskId.ValueString(),
			Settings:       settings,
		})
	}

	target := &externalEonSdkAPI.AzureVmInstanceRestoreTarget{
		Region:            config.Region.ValueString(),
		ResourceGroupName: config.ResourceGroupName.ValueString(),
		VmName:            config.VmName.ValueString(),
		VmSize:            config.VmSize.ValueString(),
		Disks:             disks,
	}
	if !config.NetworkInterface.IsNull() && config.NetworkInterface.ValueString() != "" {
		ni := config.NetworkInterface.ValueString()
		target.NetworkInterface = &ni
	}
	if !config.StartInstanceAfterRestore.IsNull() {
		v := config.StartInstanceAfterRestore.ValueBool()
		target.StartInstanceAfterRestore = &v
	}
	if tags != nil {
		target.Tags = &tags
	}

	apiReq := externalEonSdkAPI.RestoreAzureVmInstanceRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.AzureVmInstanceRestoreDestination{
			AzureVm: target,
		},
	}

	return r.client.StartAzureVmInstanceRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createAzureSqlDatabaseRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.AzureSqlConfig

	if config.Region.IsNull() || config.Region.ValueString() == "" {
		return "", fmt.Errorf("region is required for Azure SQL database restore")
	}
	if config.ResourceGroupName.IsNull() || config.ResourceGroupName.ValueString() == "" {
		return "", fmt.Errorf("resource_group_name is required for Azure SQL database restore")
	}
	if config.ServerName.IsNull() || config.ServerName.ValueString() == "" {
		return "", fmt.Errorf("server_name is required for Azure SQL database restore")
	}
	if config.AdminUserName.IsNull() || config.AdminUserName.ValueString() == "" {
		return "", fmt.Errorf("admin_user_name is required for Azure SQL database restore")
	}

	tags, err := parseMapAttribute(ctx, config.Tags)
	if err != nil {
		return "", err
	}

	target := &externalEonSdkAPI.AzureSqlDatabaseRestoreTarget{
		Region:            config.Region.ValueString(),
		ResourceGroupName: config.ResourceGroupName.ValueString(),
		ServerName:        config.ServerName.ValueString(),
		AdminUserName:     config.AdminUserName.ValueString(),
	}
	if tags != nil {
		target.Tags = &tags
	}

	apiReq := externalEonSdkAPI.RestoreAzureSqlDatabaseRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.AzureSqlDatabaseRestoreDestination{
			AzureSqlDatabase: target,
		},
	}

	return r.client.StartAzureSqlDatabaseRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createEc2InstanceRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.Ec2Config

	// Validate required fields for EC2 instance restore
	if config.Region.IsNull() || config.Region.ValueString() == "" {
		return "", fmt.Errorf("region is required for EC2 instance restore")
	}
	if config.InstanceType.IsNull() || config.InstanceType.ValueString() == "" {
		return "", fmt.Errorf("instance_type is required for EC2 instance restore")
	}
	if config.SubnetId.IsNull() || config.SubnetId.ValueString() == "" {
		return "", fmt.Errorf("subnet_id is required for EC2 instance restore")
	}
	if config.VolumeRestoreParams.IsNull() || len(config.VolumeRestoreParams.Elements()) == 0 {
		return "", fmt.Errorf("volume_restore_params is required for EC2 instance restore")
	}

	tags, err := parseMapAttribute(ctx, config.Tags)
	if err != nil {
		return "", err
	}

	var securityGroupIds []string
	if !config.SecurityGroupIds.IsNull() {
		var sgIds []types.String
		diags := config.SecurityGroupIds.ElementsAs(ctx, &sgIds, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse security group IDs")
		}
		for _, sgId := range sgIds {
			securityGroupIds = append(securityGroupIds, sgId.ValueString())
		}
	}

	var volumeParams []externalEonSdkAPI.RestoreInstanceVolumeInput
	if !config.VolumeRestoreParams.IsNull() {
		var volParams []VolumeRestoreParam
		diags := config.VolumeRestoreParams.ElementsAs(ctx, &volParams, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse volume restore parameters")
		}

		for _, volParam := range volParams {
			volumeSettings := externalEonSdkAPI.VolumeSettings{
				Type:      volParam.VolumeType.ValueString(),
				SizeBytes: volParam.VolumeSize.ValueInt64() * 1024 * 1024 * 1024, // Convert GiB to bytes
			}

			if !volParam.Iops.IsNull() {
				i32, err := SafeInt32Conversion(volParam.Iops.ValueInt64())
				if err != nil {
					return "", err
				}
				volumeSettings.Iops = &i32
			}
			if !volParam.Throughput.IsNull() {
				t32, err := SafeInt32Conversion(volParam.Throughput.ValueInt64())
				if err != nil {
					return "", err
				}
				volumeSettings.Throughput = &t32
			}

			param := externalEonSdkAPI.RestoreInstanceVolumeInput{
				ProviderVolumeId: volParam.ProviderVolumeId.ValueString(),
				VolumeSettings:   volumeSettings,
			}

			if !volParam.KmsKeyId.IsNull() && volParam.KmsKeyId.ValueString() != "" {
				param.VolumeEncryptionKeyId = volParam.KmsKeyId.ValueString()
			}

			if !volParam.Description.IsNull() && volParam.Description.ValueString() != "" {
				desc := volParam.Description.ValueString()
				param.Description = &desc
			}

			volumeParams = append(volumeParams, param)
		}
	}

	ec2Target := &externalEonSdkAPI.AwsEc2InstanceRestoreTarget{
		Region:                  config.Region.ValueString(),
		InstanceType:            config.InstanceType.ValueString(),
		SubnetId:                config.SubnetId.ValueString(),
		VolumeRestoreParameters: volumeParams,
	}

	if len(securityGroupIds) > 0 {
		ec2Target.SecurityGroupIds = securityGroupIds
	}
	if tags != nil {
		ec2Target.Tags = &tags
	}

	apiReq := externalEonSdkAPI.RestoreAwsEc2InstanceRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.AwsEc2InstanceRestoreDestination{
			AwsEc2: ec2Target,
		},
	}

	return r.client.StartEc2InstanceRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createRdsRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.RdsConfig

	// Validate required fields for RDS restore
	if config.DbInstanceIdentifier.IsNull() || config.DbInstanceIdentifier.ValueString() == "" {
		return "", fmt.Errorf("db_instance_identifier is required for RDS restore")
	}
	if config.DbInstanceClass.IsNull() || config.DbInstanceClass.ValueString() == "" {
		return "", fmt.Errorf("db_instance_class is required for RDS restore")
	}
	if config.Engine.IsNull() || config.Engine.ValueString() == "" {
		return "", fmt.Errorf("engine is required for RDS restore")
	}
	if config.Region.IsNull() || config.Region.ValueString() == "" {
		return "", fmt.Errorf("region is required for RDS restore")
	}

	if config.KmsKeyId.IsNull() || config.KmsKeyId.ValueString() == "" {
		return "", fmt.Errorf("kms_key_id is required for RDS restore")
	}

	tags, err := parseMapAttribute(ctx, config.Tags)
	if err != nil {
		return "", err
	}

	var vpcSecurityGroupIds []string
	if !config.VpcSecurityGroupIds.IsNull() {
		var sgIds []types.String
		diags := config.VpcSecurityGroupIds.ElementsAs(ctx, &sgIds, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse VPC security group IDs")
		}
		for _, sgId := range sgIds {
			vpcSecurityGroupIds = append(vpcSecurityGroupIds, sgId.ValueString())
		}
	}

	rdsTarget := &externalEonSdkAPI.AwsDatabaseDestination{
		RestoreRegion:   config.Region.ValueString(),
		RestoredName:    config.DbInstanceIdentifier.ValueString(),
		EncryptionKeyId: config.KmsKeyId.ValueString(),
	}

	if !config.SubnetGroupName.IsNull() {
		subnetGroupName := config.SubnetGroupName.ValueString()
		rdsTarget.SubnetGroup = &subnetGroupName
	}
	if len(vpcSecurityGroupIds) > 0 {
		rdsTarget.SecurityGroups = vpcSecurityGroupIds
	}
	if tags != nil {
		rdsTarget.Tags = &tags
	}

	apiReq := externalEonSdkAPI.RestoreDbToRdsInstanceRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.DatabaseDestination{
			AwsRds: rdsTarget,
		},
	}

	return r.client.StartRdsRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createS3BucketRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.S3BucketConfig

	// Validate required fields for S3 bucket restore
	if config.BucketName.IsNull() || config.BucketName.ValueString() == "" {
		return "", fmt.Errorf("bucket_name is required for S3 bucket restore")
	}

	// Build S3 restore target - use the actual SDK structure
	s3Target := &externalEonSdkAPI.S3RestoreTarget{
		BucketName: config.BucketName.ValueString(),
	}

	if !config.KeyPrefix.IsNull() {
		keyPrefix := config.KeyPrefix.ValueString()
		s3Target.Prefix = &keyPrefix
	}
	if !config.KmsKeyId.IsNull() {
		kmsKeyId := config.KmsKeyId.ValueString()
		s3Target.EncryptionKeyId = &kmsKeyId
	}

	apiReq := externalEonSdkAPI.RestoreBucketRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.ObjectStorageDestination{
			S3Bucket: s3Target,
		},
	}

	return r.client.StartS3BucketRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createS3FileRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.S3FileConfig

	// Validate required fields for S3 file restore
	if config.BucketName.IsNull() || config.BucketName.ValueString() == "" {
		return "", fmt.Errorf("bucket_name is required for S3 file restore")
	}
	if config.Files.IsNull() || len(config.Files.Elements()) == 0 {
		return "", fmt.Errorf("files is required for S3 file restore")
	}

	var files []externalEonSdkAPI.FilePath
	if !config.Files.IsNull() {
		var fileList []FileRestoreParam
		diags := config.Files.ElementsAs(ctx, &fileList, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse files list")
		}

		for _, file := range fileList {
			filePath := externalEonSdkAPI.FilePath{
				Path: file.Path.ValueString(),
			}
			if !file.IsDirectory.IsNull() {
				filePath.IsDirectory = file.IsDirectory.ValueBool()
			} else {
				filePath.IsDirectory = false
			}
			files = append(files, filePath)
		}
	}

	s3Target := &externalEonSdkAPI.S3RestoreTarget{
		BucketName: config.BucketName.ValueString(),
	}

	if !config.KeyPrefix.IsNull() {
		keyPrefix := config.KeyPrefix.ValueString()
		s3Target.Prefix = &keyPrefix
	}
	if !config.KmsKeyId.IsNull() {
		kmsKeyId := config.KmsKeyId.ValueString()
		s3Target.EncryptionKeyId = &kmsKeyId
	}

	apiReq := externalEonSdkAPI.RestoreFilesRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Files:            files,
		Destination: externalEonSdkAPI.ObjectStorageDestination{
			S3Bucket: s3Target,
		},
	}

	return r.client.StartS3FileRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

// GCP restore methods

func (r *RestoreJobResource) createGcpVmInstanceRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.GcpVmConfig

	if config.Zone.IsNull() || config.Zone.ValueString() == "" {
		return "", fmt.Errorf("zone is required for GCP VM instance restore")
	}
	if config.MachineType.IsNull() || config.MachineType.ValueString() == "" {
		return "", fmt.Errorf("machine_type is required for GCP VM instance restore")
	}
	if config.Name.IsNull() || config.Name.ValueString() == "" {
		return "", fmt.Errorf("name is required for GCP VM instance restore")
	}
	if config.NetworkName.IsNull() || config.NetworkName.ValueString() == "" {
		return "", fmt.Errorf("network_name is required for GCP VM instance restore")
	}
	if config.SubnetName.IsNull() || config.SubnetName.ValueString() == "" {
		return "", fmt.Errorf("subnet_name is required for GCP VM instance restore")
	}

	labels, err := parseMapAttribute(ctx, config.Labels)
	if err != nil {
		return "", err
	}

	// Parse disks
	var diskInputs []externalEonSdkAPI.RestoreGcpInstanceDiskInput
	var diskParams []GcpDiskRestoreParam
	diags := config.Disks.ElementsAs(ctx, &diskParams, false)
	if diags.HasError() {
		return "", fmt.Errorf("failed to parse disks")
	}

	for _, dp := range diskParams {
		diskSettings := externalEonSdkAPI.GcpDiskSettings{
			Name:      dp.Name.ValueString(),
			Type:      dp.DiskType.ValueString(),
			SizeBytes: dp.SizeBytes.ValueInt64(),
		}

		if !dp.Iops.IsNull() {
			iops := dp.Iops.ValueInt64()
			diskSettings.Iops = &iops
		}
		if !dp.Throughput.IsNull() {
			throughput := dp.Throughput.ValueInt64()
			diskSettings.Throughput = &throughput
		}
		if !dp.Description.IsNull() && dp.Description.ValueString() != "" {
			desc := dp.Description.ValueString()
			diskSettings.Description = &desc
		}
		diskLabels, parseErr := parseMapAttribute(ctx, dp.Labels)
		if parseErr != nil {
			return "", parseErr
		}
		if diskLabels != nil {
			diskSettings.Labels = &diskLabels
		}
		if !dp.EncryptionKeyId.IsNull() && dp.EncryptionKeyId.ValueString() != "" {
			keyId := dp.EncryptionKeyId.ValueString()
			diskSettings.EncryptionKeyId = &keyId
		}

		diskInputs = append(diskInputs, externalEonSdkAPI.RestoreGcpInstanceDiskInput{
			ProviderDiskId: dp.ProviderDiskId.ValueString(),
			Settings:       diskSettings,
		})
	}

	vmTarget := &externalEonSdkAPI.GcpVmInstanceRestoreTarget{
		Zone:        config.Zone.ValueString(),
		MachineType: config.MachineType.ValueString(),
		Name:        config.Name.ValueString(),
		NetworkName: config.NetworkName.ValueString(),
		SubnetName:  config.SubnetName.ValueString(),
		Disks:       diskInputs,
	}

	if !config.NetworkHostProject.IsNull() && config.NetworkHostProject.ValueString() != "" {
		nhp := config.NetworkHostProject.ValueString()
		vmTarget.NetworkHostProject = &nhp
	}
	if labels != nil {
		vmTarget.Labels = &labels
	}
	if !config.StartInstanceAfterRestore.IsNull() {
		startAfterRestore := config.StartInstanceAfterRestore.ValueBool()
		vmTarget.StartInstanceAfterRestore = &startAfterRestore
	}

	apiReq := externalEonSdkAPI.RestoreGcpVmInstanceRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.GcpVmInstanceRestoreDestination{
			GcpVm: vmTarget,
		},
	}

	return r.client.StartGcpVmInstanceRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createGcpDiskRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.GcpDiskConfig

	if config.ProviderDiskId.IsNull() || config.ProviderDiskId.ValueString() == "" {
		return "", fmt.Errorf("provider_disk_id is required for GCP disk restore")
	}
	if config.Zone.IsNull() || config.Zone.ValueString() == "" {
		return "", fmt.Errorf("zone is required for GCP disk restore")
	}
	if config.Name.IsNull() || config.Name.ValueString() == "" {
		return "", fmt.Errorf("name is required for GCP disk restore")
	}
	if config.DiskType.IsNull() || config.DiskType.ValueString() == "" {
		return "", fmt.Errorf("disk_type is required for GCP disk restore")
	}
	if config.SizeBytes.IsNull() || config.SizeBytes.ValueInt64() == 0 {
		return "", fmt.Errorf("size_bytes is required for GCP disk restore")
	}

	diskSettings := externalEonSdkAPI.GcpDiskSettings{
		Name:      config.Name.ValueString(),
		Type:      config.DiskType.ValueString(),
		SizeBytes: config.SizeBytes.ValueInt64(),
	}

	if !config.Iops.IsNull() {
		iops := config.Iops.ValueInt64()
		diskSettings.Iops = &iops
	}
	if !config.Throughput.IsNull() {
		throughput := config.Throughput.ValueInt64()
		diskSettings.Throughput = &throughput
	}
	if !config.Description.IsNull() && config.Description.ValueString() != "" {
		desc := config.Description.ValueString()
		diskSettings.Description = &desc
	}
	diskLabels, err := parseMapAttribute(ctx, config.Labels)
	if err != nil {
		return "", err
	}
	if diskLabels != nil {
		diskSettings.Labels = &diskLabels
	}
	if !config.EncryptionKeyId.IsNull() && config.EncryptionKeyId.ValueString() != "" {
		keyId := config.EncryptionKeyId.ValueString()
		diskSettings.EncryptionKeyId = &keyId
	}

	diskTarget := &externalEonSdkAPI.GcpDiskTarget{
		Zone:     config.Zone.ValueString(),
		Settings: diskSettings,
	}

	apiReq := externalEonSdkAPI.RestoreGcpDiskRequest{
		ProviderDiskId:   config.ProviderDiskId.ValueString(),
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.GcpDiskRestoreDestination{
			GcpDisk: diskTarget,
		},
	}

	return r.client.StartGcpDiskRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createGcpCloudSqlRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.GcpCloudSqlConfig

	if config.Zone.IsNull() || config.Zone.ValueString() == "" {
		return "", fmt.Errorf("zone is required for GCP Cloud SQL restore")
	}
	if config.Name.IsNull() || config.Name.ValueString() == "" {
		return "", fmt.Errorf("name is required for GCP Cloud SQL restore")
	}
	if config.NetworkType.IsNull() || config.NetworkType.ValueString() == "" {
		return "", fmt.Errorf("network_type is required for GCP Cloud SQL restore")
	}

	networkType, err := externalEonSdkAPI.NewGcpNetworkTypeFromValue(config.NetworkType.ValueString())
	if err != nil {
		return "", fmt.Errorf("invalid network_type: %s. Valid values are: PUBLIC, PRIVATE", config.NetworkType.ValueString())
	}

	sqlTarget := &externalEonSdkAPI.GcpCloudSqlTarget{
		Zone:        config.Zone.ValueString(),
		Name:        config.Name.ValueString(),
		NetworkType: *networkType,
	}

	if !config.NetworkName.IsNull() && config.NetworkName.ValueString() != "" {
		nn := config.NetworkName.ValueString()
		sqlTarget.NetworkName = &nn
	}
	if !config.NetworkHostProject.IsNull() && config.NetworkHostProject.ValueString() != "" {
		nhp := config.NetworkHostProject.ValueString()
		sqlTarget.NetworkHostProject = &nhp
	}
	sqlLabels, err := parseMapAttribute(ctx, config.Labels)
	if err != nil {
		return "", err
	}
	if sqlLabels != nil {
		sqlTarget.Labels = &sqlLabels
	}

	nullableSqlTarget := externalEonSdkAPI.NewNullableGcpCloudSqlTarget(sqlTarget)
	apiReq := externalEonSdkAPI.RestoreGcpCloudSqlRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.GcpCloudSqlRestoreDestination{
			GcpCloudSql: *nullableSqlTarget,
		},
	}

	return r.client.StartGcpCloudSqlRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createGcsBucketRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.GcsBucketConfig

	// Validate required fields
	if config.BucketName.IsNull() || config.BucketName.ValueString() == "" {
		return "", fmt.Errorf("bucket_name is required for GCS bucket restore")
	}

	gcsTarget := &externalEonSdkAPI.GCSRestoreTarget{
		BucketName: config.BucketName.ValueString(),
	}

	if !config.KeyPrefix.IsNull() {
		prefix := config.KeyPrefix.ValueString()
		gcsTarget.Prefix = &prefix
	}

	apiReq := externalEonSdkAPI.RestoreBucketRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.ObjectStorageDestination{
			GcsBucket: gcsTarget,
		},
	}

	return r.client.StartS3BucketRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) createGcsFileRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.GcsFileConfig

	// Validate required fields
	if config.BucketName.IsNull() || config.BucketName.ValueString() == "" {
		return "", fmt.Errorf("bucket_name is required for GCS file restore")
	}
	if config.Files.IsNull() || len(config.Files.Elements()) == 0 {
		return "", fmt.Errorf("files is required for GCS file restore")
	}

	var files []externalEonSdkAPI.FilePath
	var fileList []FileRestoreParam
	diags := config.Files.ElementsAs(ctx, &fileList, false)
	if diags.HasError() {
		return "", fmt.Errorf("failed to parse files list")
	}

	for _, file := range fileList {
		filePath := externalEonSdkAPI.FilePath{
			Path: file.Path.ValueString(),
		}
		if !file.IsDirectory.IsNull() {
			filePath.IsDirectory = file.IsDirectory.ValueBool()
		} else {
			filePath.IsDirectory = false
		}
		files = append(files, filePath)
	}

	gcsTarget := &externalEonSdkAPI.GCSRestoreTarget{
		BucketName: config.BucketName.ValueString(),
	}

	if !config.KeyPrefix.IsNull() {
		prefix := config.KeyPrefix.ValueString()
		gcsTarget.Prefix = &prefix
	}

	apiReq := externalEonSdkAPI.RestoreFilesRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Files:            files,
		Destination: externalEonSdkAPI.ObjectStorageDestination{
			GcsBucket: gcsTarget,
		},
	}

	return r.client.StartS3FileRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

// BigQuery restore methods

func (r *RestoreJobResource) createGcpBigQueryDatasetRestore(ctx context.Context, data RestoreJobResourceModel, resourceId string) (string, error) {
	config := data.GcpBigQueryDatasetConfig

	if config.DatasetId.IsNull() || config.DatasetId.ValueString() == "" {
		return "", fmt.Errorf("dataset_id is required for BigQuery dataset restore")
	}
	if config.Location.IsNull() || config.Location.ValueString() == "" {
		return "", fmt.Errorf("location is required for BigQuery dataset restore")
	}

	apiReq := client.BigQueryRestoreRequest{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: client.BigQueryRestoreDestination{
			DatasetId: config.DatasetId.ValueString(),
			Location:  config.Location.ValueString(),
		},
	}

	// Optional table filter
	if !config.Tables.IsNull() && len(config.Tables.Elements()) > 0 {
		var tableParams []GcpBigQueryTableParam
		diags := config.Tables.ElementsAs(ctx, &tableParams, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse tables list")
		}

		var tables []string
		for _, tp := range tableParams {
			if !tp.TableId.IsNull() && tp.TableId.ValueString() != "" {
				tables = append(tables, tp.TableId.ValueString())
			}
		}
		if len(tables) > 0 {
			apiReq.Tables = tables
		}
	}

	return r.client.StartBigQueryDatasetRestore(ctx, resourceId, data.SnapshotId.ValueString(), apiReq)
}

func (r *RestoreJobResource) updateJobStatus(ctx context.Context, data *RestoreJobResourceModel, job *externalEonSdkAPI.RestoreJob) {
	data.Status = types.StringValue(string(job.GetJobExecutionDetails().Status))
	data.CreatedAt = types.StringValue(job.GetJobExecutionDetails().CreatedTime.Format(time.RFC3339))

	if job.GetJobExecutionDetails().StatusMessage != nil {
		data.StatusMessage = types.StringValue(*job.GetJobExecutionDetails().StatusMessage)
	} else {
		data.StatusMessage = types.StringNull()
	}

	if job.GetJobExecutionDetails().StartTime.IsSet() {
		data.StartedAt = types.StringValue(job.GetJobExecutionDetails().StartTime.Get().Format(time.RFC3339))
	} else {
		data.StartedAt = types.StringNull()
	}

	if job.GetJobExecutionDetails().EndTime.IsSet() {
		data.CompletedAt = types.StringValue(job.GetJobExecutionDetails().EndTime.Get().Format(time.RFC3339))
	} else {
		data.CompletedAt = types.StringNull()
	}

	if job.GetJobExecutionDetails().DurationSeconds.IsSet() {
		data.DurationSeconds = types.Int64Value(*job.GetJobExecutionDetails().DurationSeconds.Get())
	} else {
		data.DurationSeconds = types.Int64Null()
	}
}

func (r *RestoreJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RestoreJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobID := data.JobId.ValueString()
	if jobID == "" {
		jobID = data.Id.ValueString()
	}
	if jobID == "" {
		return
	}

	job, err := r.client.GetRestoreJob(ctx, jobID)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore job: %s", err))
		return
	}

	data.JobId = types.StringValue(job.GetJobExecutionDetails().JobId)
	data.Id = types.StringValue(job.GetJobExecutionDetails().JobId)
	r.updateJobStatus(ctx, &data, job)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RestoreJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestoreJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RestoreJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Restore job removed from state", map[string]interface{}{"job_id": data.JobId.ValueString()})
}

func (r *RestoreJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("job_id"), req.ID)...)
}
