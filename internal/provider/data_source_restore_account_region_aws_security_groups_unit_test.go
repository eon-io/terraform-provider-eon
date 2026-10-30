package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestoreAccountRegionAwsSecurityGroupsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsSecurityGroupsDataSource())
}
