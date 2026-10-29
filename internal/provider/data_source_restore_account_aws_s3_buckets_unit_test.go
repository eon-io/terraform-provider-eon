package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestoreAccountAwsS3BucketsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountAwsS3BucketsDataSource())
}
