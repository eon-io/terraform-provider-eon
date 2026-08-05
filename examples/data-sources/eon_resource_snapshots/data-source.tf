# Example: List restorable snapshots for an inventory resource
data "eon_resource_snapshots" "demo" {
  resource_id = "1ee34dc5-0a7c-4e56-a820-917371e05c8d"
}

# Example: Narrow snapshots to a point-in-time window
data "eon_resource_snapshots" "recent" {
  resource_id              = "1ee34dc5-0a7c-4e56-a820-917371e05c8d"
  point_in_time_start_date = "2024-01-01"
  point_in_time_end_date   = "2024-12-31"
}

output "latest_snapshot_id" {
  description = "First listed snapshot ID for the resource"
  value       = try(data.eon_resource_snapshots.demo.snapshots[0].id, null)
}
