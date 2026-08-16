package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionApprovalRulesDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewActionApprovalRulesDataSource())
}
