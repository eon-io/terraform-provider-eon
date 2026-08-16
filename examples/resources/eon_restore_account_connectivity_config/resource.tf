# The connectivity config attaches to a restore account
resource "eon_restore_account" "aws_disaster_recovery" {
  name           = "Disaster Recovery AWS Account"
  cloud_provider = "AWS"

  aws {
    role_arn = "arn:aws:iam::555666777888:role/EonRestoreRole"
  }
}

resource "eon_restore_account" "gcp_disaster_recovery" {
  name           = "Disaster Recovery GCP Project"
  cloud_provider = "GCP"

  gcp {
    project_id      = "my-gcp-project-id"
    service_account = "eon-restore@my-gcp-project-id.iam.gserviceaccount.com"
  }
}

# Example: Configure connectivity for an AWS restore account
resource "eon_restore_account_connectivity_config" "aws" {
  restore_account_id = eon_restore_account.aws_disaster_recovery.id

  aws {
    vpc_configs {
      region = "us-east-1"
      vpc    = "vpc-01234567890123456"

      subnets_per_availability_zone {
        availability_zone = "us-east-1a"
        subnet_id         = "subnet-01234567890123456"
      }

      security_groups {
        restore_server        = ["sg-0d447a77aa14f9e90"]
        restored_rds_instance = ["sg-02ce123456e7893c7"]
      }
    }
  }
}

# Example: Configure connectivity for a GCP restore account using a shared VPC
resource "eon_restore_account_connectivity_config" "gcp" {
  restore_account_id = eon_restore_account.gcp_disaster_recovery.id

  gcp {
    network_configs {
      network              = "my-network"
      is_shared_vpc        = true
      network_host_project = "my-shared-vpc-project"

      subnets_per_region {
        region      = "us-east1"
        subnet_name = "my-subnet"
      }
    }
  }
}
