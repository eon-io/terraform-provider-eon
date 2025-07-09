package provider

import (
	"errors"
	"math"
)

// Helper functions for safe int64 to int32 conversion with bounds checking

// safeInt32Conversion performs bounds checking for int64 to int32 conversion
func SafeInt32Conversion(value int64) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return int32(value), nil
	}
	return 0, errors.New("value out of int32 bounds")
}
