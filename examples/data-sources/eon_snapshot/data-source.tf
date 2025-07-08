data "eon_snapshot" "s3_snapshot" {
  id = "18618b5a-c467-4f19-acf5-b31c63ba865b"
}

output "s3_snapshot_info" {
  value = {
    id            = data.eon_snapshot.s3_snapshot.id
    resource_id   = data.eon_snapshot.s3_snapshot.resource_id
    vault_id      = data.eon_snapshot.s3_snapshot.vault_id
    created_at    = data.eon_snapshot.s3_snapshot.created_at
    point_in_time = data.eon_snapshot.s3_snapshot.point_in_time
  }
}