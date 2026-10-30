package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestoreAccountRegionAwsInstanceTypesDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsInstanceTypesDataSource())
}
