#1. EBS Volume Restore
resource "eon_restore_job" "ebs_volume" {
  restore_type        = "partial"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 120
  wait_for_completion = true

  ebs_config {
    provider_volume_id       = "vol-0f55f55a02e069c53"
    availability_zone        = "us-east-1a"
    volume_type              = "gp3"
    volume_size              = 100
    description              = "Test EBS volume restore from Eon using new resource"
    volume_encryption_key_id = "alias/aws/ebs"

    tags = {
      Name = "eon-test-restore"
      Test = "true"
    }
  }
}

# 2. EC2 Instance Restore
resource "eon_restore_job" "ec2_instance" {
  restore_type        = "full"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 120
  wait_for_completion = true

  ec2_config {
    region        = "us-east-1"
    instance_type = "t3.medium"
    subnet_id     = "subnet-0123456789abcdef0"
    security_group_ids = [
      "sg-0123456789abcdef0",
      "sg-0987654321fedcba0"
    ]

    tags = {
      Name        = "eon-restored-instance"
      Environment = "test"
      RestoreJob  = "true"
    }

    volume_restore_params {
      provider_volume_id = "vol-0f55f55a02e069c53"
      volume_type        = "gp3"
      volume_size        = 20
      iops               = 3000
      description        = "Root volume"
      kms_key_id         = "arn:aws:kms:us-east-1:851725316996:key/20c82703-ea74-45f9-a38c-0c142023d694"
    }


  }
}

# 3. RDS Database Restore
resource "eon_restore_job" "rds_database" {
  restore_type        = "full"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 180
  wait_for_completion = true

  rds_config {
    db_instance_identifier = "eon-restored-db"
    db_instance_class      = "db.t3.micro"
    engine                 = "mysql"
    region                 = "us-east-1"
    subnet_group_name      = "default"
    vpc_security_group_ids = [
      "sg-0123456789abcdef0"
    ]
    allocated_storage       = 20
    storage_type            = "gp2"
    backup_retention_period = 7
    multi_az                = false
    publicly_accessible     = false
    storage_encrypted       = true
    kms_key_id              = "alias/aws/rds"

    tags = {
      Name        = "eon-restored-database"
      Environment = "test"
      RestoreJob  = "true"
    }
  }
}
#
# 4. S3 Bucket Restore
resource "eon_restore_job" "s3_bucket" {
  restore_type        = "full"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 90
  wait_for_completion = true

  s3_bucket_config {
    bucket_name = "my-bucket"
    key_prefix  = "restored-data/"
  }
}

# 5. S3 File Restore
resource "eon_restore_job" "s3_files" {
  restore_type        = "partial"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 60
  wait_for_completion = true

  s3_file_config {
    bucket_name = "my-bucket"
    key_prefix  = "restored-files/"

    files {
      path         = "my-file.yml"
      is_directory = false
    }

    files {
      path         = "my-other-file.yaml"
      is_directory = false
    }
  }
}


# 6. GCP VM Instance Restore
resource "eon_restore_job" "gcp_vm" {
  restore_type        = "full"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 120
  wait_for_completion = true

  gcp_vm_config {
    zone         = "us-central1-a"
    machine_type = "e2-medium"
    name         = "eon-restored-vm"
    network_name = "default"
    subnet_name  = "default"

    start_instance_after_restore = true

    labels = {
      environment = "test"
      restored    = "true"
    }

    disks {
      provider_disk_id = "1234567890123456789"
      name             = "eon-restored-boot-disk"
      disk_type        = "pd-balanced"
      size_bytes       = 10737418240
      description      = "Boot disk"
    }
  }
}

# 7. GCP Disk Restore (standalone disk or partial VM restore)
resource "eon_restore_job" "gcp_disk" {
  restore_type        = "partial"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 60
  wait_for_completion = true

  gcp_disk_config {
    provider_disk_id = "1234567890123456789"
    zone             = "us-central1-a"
    name             = "eon-restored-disk"
    disk_type        = "pd-ssd"
    size_bytes       = 10737418240
    description      = "Restored data disk"

    labels = {
      environment = "test"
      restored    = "true"
    }
  }
}

# 8. GCP Cloud SQL Restore
resource "eon_restore_job" "gcp_cloud_sql" {
  restore_type        = "full"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 180
  wait_for_completion = true

  gcp_cloud_sql_config {
    zone         = "us-central1-a"
    name         = "eon-restored-sql-instance"
    network_type = "PUBLIC"

    labels = {
      environment = "test"
      restored    = "true"
    }
  }
}

# 9. GCS Bucket Restore
resource "eon_restore_job" "gcs_bucket" {
  restore_type        = "full"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 90
  wait_for_completion = true

  gcs_bucket_config {
    bucket_name = "my-gcs-bucket"
    key_prefix  = "restored-data/"
  }
}

# 10. GCS File Restore
resource "eon_restore_job" "gcs_files" {
  restore_type        = "partial"
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 60
  wait_for_completion = true

  gcs_file_config {
    bucket_name = "my-gcs-bucket"
    key_prefix  = "restored-files/"

    files {
      path         = "my-file.yml"
      is_directory = false
    }

    files {
      path         = "my-directory/"
      is_directory = true
    }
  }
}

# 11. GCP BigQuery Dataset Restore (all tables)
resource "eon_restore_job" "bigquery_dataset_full" {
  restore_type        = "full" # any value accepted for BigQuery
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 120
  wait_for_completion = true

  gcp_bigquery_restore_dataset_config {
    dataset_id = "my_dataset_restored"
    location   = "US"
  }
}

# 12. GCP BigQuery Dataset Restore (filtered by specific tables)
resource "eon_restore_job" "bigquery_dataset_filtered" {
  restore_type        = "full" # any value accepted for BigQuery
  snapshot_id         = "cd6312c7-0713-4a24-a0b3-b838c1108d2f"
  restore_account_id  = "e696c7f0-17c6-4d9b-b589-591293c00d36"
  timeout_minutes     = 120
  wait_for_completion = true

  gcp_bigquery_restore_dataset_config {
    dataset_id = "my_dataset_partial_restore"
    location   = "US"

    tables {
      table_id = "my_table_1"
    }

    tables {
      table_id = "my_table_2"
    }
  }
}

output "ebs_restore_info" {
  value = {
    job_id       = eon_restore_job.ebs_volume.job_id
    status       = eon_restore_job.ebs_volume.status
    created_at   = eon_restore_job.ebs_volume.created_at
    completed_at = eon_restore_job.ebs_volume.completed_at
  }
}

output "ec2_restore_info" {
  value = {
    job_id       = eon_restore_job.ec2_instance.job_id
    status       = eon_restore_job.ec2_instance.status
    created_at   = eon_restore_job.ec2_instance.created_at
    completed_at = eon_restore_job.ec2_instance.completed_at
  }
}

output "rds_restore_info" {
  value = {
    job_id       = eon_restore_job.rds_database.job_id
    status       = eon_restore_job.rds_database.status
    created_at   = eon_restore_job.rds_database.created_at
    completed_at = eon_restore_job.rds_database.completed_at
  }
}

output "s3_bucket_restore_info" {
  value = {
    job_id       = eon_restore_job.s3_bucket.job_id
    status       = eon_restore_job.s3_bucket.status
    created_at   = eon_restore_job.s3_bucket.created_at
    completed_at = eon_restore_job.s3_bucket.completed_at
  }
}

output "s3_files_restore_info" {
  value = {
    job_id       = eon_restore_job.s3_files.job_id
    status       = eon_restore_job.s3_files.status
    created_at   = eon_restore_job.s3_files.created_at
    completed_at = eon_restore_job.s3_files.completed_at
  }
}

output "gcp_vm_restore_info" {
  value = {
    job_id       = eon_restore_job.gcp_vm.job_id
    status       = eon_restore_job.gcp_vm.status
    created_at   = eon_restore_job.gcp_vm.created_at
    completed_at = eon_restore_job.gcp_vm.completed_at
  }
}

output "gcp_disk_restore_info" {
  value = {
    job_id       = eon_restore_job.gcp_disk.job_id
    status       = eon_restore_job.gcp_disk.status
    created_at   = eon_restore_job.gcp_disk.created_at
    completed_at = eon_restore_job.gcp_disk.completed_at
  }
}

output "gcp_cloud_sql_restore_info" {
  value = {
    job_id       = eon_restore_job.gcp_cloud_sql.job_id
    status       = eon_restore_job.gcp_cloud_sql.status
    created_at   = eon_restore_job.gcp_cloud_sql.created_at
    completed_at = eon_restore_job.gcp_cloud_sql.completed_at
  }
}

output "gcs_bucket_restore_info" {
  value = {
    job_id       = eon_restore_job.gcs_bucket.job_id
    status       = eon_restore_job.gcs_bucket.status
    created_at   = eon_restore_job.gcs_bucket.created_at
    completed_at = eon_restore_job.gcs_bucket.completed_at
  }
}

output "gcs_files_restore_info" {
  value = {
    job_id       = eon_restore_job.gcs_files.job_id
    status       = eon_restore_job.gcs_files.status
    created_at   = eon_restore_job.gcs_files.created_at
    completed_at = eon_restore_job.gcs_files.completed_at
  }
}

output "bigquery_dataset_full_restore_info" {
  value = {
    job_id       = eon_restore_job.bigquery_dataset_full.job_id
    status       = eon_restore_job.bigquery_dataset_full.status
    created_at   = eon_restore_job.bigquery_dataset_full.created_at
    completed_at = eon_restore_job.bigquery_dataset_full.completed_at
  }
}

output "bigquery_dataset_filtered_restore_info" {
  value = {
    job_id       = eon_restore_job.bigquery_dataset_filtered.job_id
    status       = eon_restore_job.bigquery_dataset_filtered.status
    created_at   = eon_restore_job.bigquery_dataset_filtered.created_at
    completed_at = eon_restore_job.bigquery_dataset_filtered.completed_at
  }
}
