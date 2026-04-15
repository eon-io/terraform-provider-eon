package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// alwaysConnectedModifier is a plan modifier that always plans "CONNECTED"
// for the status field. This causes Terraform to detect drift when the API
// reports a non-CONNECTED status and trigger an Update (reconnect).
type alwaysConnectedModifier struct{}

func (m alwaysConnectedModifier) Description(_ context.Context) string {
	return "Plans status as CONNECTED so that a disconnected account triggers an update (reconnect)."
}

func (m alwaysConnectedModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m alwaysConnectedModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
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

// AlwaysConnected returns a plan modifier that forces the planned value to "CONNECTED".
func AlwaysConnected() planmodifier.String {
	return alwaysConnectedModifier{}
}
