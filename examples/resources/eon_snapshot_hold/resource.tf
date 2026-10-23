# Look up the snapshot to hold
data "eon_snapshot" "latest" {
  id = "18618b5a-c467-4f19-acf5-b31c63ba865b"
}

# Hold a snapshot so retention policy cannot delete it
resource "eon_snapshot_hold" "compliance" {
  snapshot_id = data.eon_snapshot.latest.id
  description = "Held for compliance investigation"
}
