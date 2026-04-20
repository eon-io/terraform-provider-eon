package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// reconnectIfDisconnectedModifier is a plan modifier that flips the planned
// status to "CONNECTED" when the state reports "DISCONNECTED". This causes
// Terraform to detect drift and trigger an Update (which performs a reconnect).
// Other non-CONNECTED states (e.g. INSUFFICIENT_PERMISSIONS) are preserved
// as-is since they require manual intervention.
type reconnectIfDisconnectedModifier struct{}

func (m reconnectIfDisconnectedModifier) Description(_ context.Context) string {
	return "Plans status as CONNECTED when the current state is DISCONNECTED so that the next apply triggers a reconnect."
}

func (m reconnectIfDisconnectedModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m reconnectIfDisconnectedModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// On create the state is nil — let the API populate the initial value.
	if req.State.Raw.IsNull() {
		return
	}

	// Only plan a change when the account is DISCONNECTED.
	// Other non-CONNECTED states (e.g. INSUFFICIENT_PERMISSIONS) require
	// manual intervention and should not trigger an automatic reconnect.
	if req.StateValue.ValueString() == "DISCONNECTED" {
		resp.PlanValue = types.StringValue("CONNECTED")
		return
	}

	// Preserve the current state value for all other statuses.
	resp.PlanValue = req.StateValue
}

// ReconnectOnDisconnected returns a plan modifier that flips the planned
// status to "CONNECTED" when the state is "DISCONNECTED", triggering an
// Update on the next apply so the provider can attempt a reconnect.
func ReconnectOnDisconnected() planmodifier.String {
	return reconnectIfDisconnectedModifier{}
}
