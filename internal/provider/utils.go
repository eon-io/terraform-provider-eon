package provider

import (
	"fmt"
	"math"
)

// SafeInt32Conversion performs bounds checking for int64 to int32 conversion
func SafeInt32Conversion(value int64) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("value %d out of int32 bounds", value)
	}
	return int32(value), nil
}
