package provider

import (
	"context"
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

	// Restore type specific configuration blocks
	EbsConfig      *EbsRestoreConfig      `tfsdk:"ebs_config"`
	Ec2Config      *Ec2RestoreConfig      `tfsdk:"ec2_config"`
	RdsConfig      *RdsRestoreConfig      `tfsdk:"rds_config"`
	S3BucketConfig *S3BucketRestoreConfig `tfsdk:"s3_bucket_config"`
	S3FileConfig   *S3FileRestoreConfig   `tfsdk:"s3_file_config"`

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

type S3FileParam struct {
	Path        types.String `tfsdk:"path"`
	IsDirectory types.Bool   `tfsdk:"is_directory"`
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
				MarkdownDescription: "Type of restore job: `full` for full resource restore, `partial` for partial restore.",
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
	if restoreType != "full" && restoreType != "partial" {
		resp.Diagnostics.AddError("Configuration Error", fmt.Sprintf("Invalid restore_type: %s. Supported types: full, partial", restoreType))
		return
	}
	var jobId string

	// Fallback to inventory-based detection for backward compatibility
	switch inventoryResource.GetResourceType() {
	case externalEonSdkAPI.AWS_EC2:
		if restoreType == "partial" {
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
	default:
		resp.Diagnostics.AddError("Configuration Error", fmt.Sprintf("Unsupported resource type: %s. Supported types: AWS_EC2, AWS_RDS, AWS_S3. Please provide one of: ebs_config, ec2_config, rds_config, s3_bucket_config, or s3_file_config", inventoryResource.GetResourceType()))
		return
	}

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

	var tags map[string]string
	if !config.Tags.IsNull() {
		tagsMap := make(map[string]types.String, len(config.Tags.Elements()))
		diags := config.Tags.ElementsAs(ctx, &tagsMap, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse tags")
		}
		tags = make(map[string]string)
		for k, v := range tagsMap {
			tags[k] = v.ValueString()
		}
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

	var tags map[string]string
	if !config.Tags.IsNull() {
		tagsMap := make(map[string]types.String, len(config.Tags.Elements()))
		diags := config.Tags.ElementsAs(ctx, &tagsMap, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse tags")
		}
		tags = make(map[string]string)
		for k, v := range tagsMap {
			tags[k] = v.ValueString()
		}
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

	ec2Target := &externalEonSdkAPI.Ec2InstanceRestoreTarget{
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

	apiReq := externalEonSdkAPI.RestoreInstanceInput{
		RestoreAccountId: data.RestoreAccountId.ValueString(),
		Destination: externalEonSdkAPI.Ec2InstanceRestoreDestination{
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

	var tags map[string]string
	if !config.Tags.IsNull() {
		tagsMap := make(map[string]types.String, len(config.Tags.Elements()))
		diags := config.Tags.ElementsAs(ctx, &tagsMap, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse tags")
		}
		tags = make(map[string]string)
		for k, v := range tagsMap {
			tags[k] = v.ValueString()
		}
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
		var fileList []S3FileParam
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

	if data.JobId.IsNull() || data.JobId.ValueString() == "" {
		return
	}

	job, err := r.client.GetRestoreJob(ctx, data.JobId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore job: %s", err))
		return
	}

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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
