# Hold a snapshot so retention policy cannot delete it
resource "eon_snapshot_hold" "compliance" {
  snapshot_id = data.eon_snapshot.latest.id
  description = "Held for compliance investigation"
}
