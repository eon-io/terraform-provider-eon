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