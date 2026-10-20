package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLabelsToSet(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		wantNull bool
		want     []string
	}{
		{name: "nil becomes null", labels: nil, wantNull: true},
		{name: "empty becomes null", labels: []string{}, wantNull: true},
		{name: "values are preserved", labels: []string{"production", "tier-1"}, want: []string{"production", "tier-1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := labelsToSet(test.labels)
			if got.IsNull() != test.wantNull {
				t.Fatalf("IsNull() = %v, want %v", got.IsNull(), test.wantNull)
			}
			if test.wantNull {
				return
			}
			if roundTripped := labelsFromSet(got); !labelsEqual(roundTripped, test.want) {
				t.Fatalf("round trip = %v, want %v", roundTripped, test.want)
			}
		})
	}
}

func TestLabelsFromSet(t *testing.T) {
	t.Run("null yields nil", func(t *testing.T) {
		if got := labelsFromSet(types.SetNull(types.StringType)); got != nil {
			t.Fatalf("got %v, want nil", got)
		}
	})

	t.Run("unknown yields nil", func(t *testing.T) {
		if got := labelsFromSet(types.SetUnknown(types.StringType)); got != nil {
			t.Fatalf("got %v, want nil", got)
		}
	})

	t.Run("null elements are dropped", func(t *testing.T) {
		set := types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("production"),
			types.StringNull(),
		})
		if got := labelsFromSet(set); !labelsEqual(got, []string{"production"}) {
			t.Fatalf("got %v, want [production]", got)
		}
	})
}

func TestLabelsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{name: "both empty", a: nil, b: []string{}, want: true},
		{name: "same order", a: []string{"a", "b"}, b: []string{"a", "b"}, want: true},
		{name: "different order still equal", a: []string{"b", "a"}, b: []string{"a", "b"}, want: true},
		{name: "different lengths", a: []string{"a"}, b: []string{"a", "b"}, want: false},
		{name: "different values", a: []string{"a"}, b: []string{"b"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := labelsEqual(test.a, test.b); got != test.want {
				t.Fatalf("labelsEqual(%v, %v) = %v, want %v", test.a, test.b, got, test.want)
			}
		})
	}
}
