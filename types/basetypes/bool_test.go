package basetypes

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestBoolTypeValueFromTerraform(t *testing.T) {
	t.Parallel()

	type testCase struct {
		input       tftypes.Value
		expectation attr.Value
		expectedErr string
	}
	tests := map[string]testCase{
		"true": {
			input:       tftypes.NewValue(tftypes.Bool, true),
			expectation: NewBoolValue(true),
		},
		"false": {
			input:       tftypes.NewValue(tftypes.Bool, false),
			expectation: NewBoolValue(false),
		},
		"unknown": {
			input:       tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
			expectation: NewBoolUnknown(),
		},
		"null": {
			input:       tftypes.NewValue(tftypes.Bool, nil),
			expectation: NewBoolNull(),
		},
		"wrongType": {
			input:       tftypes.NewValue(tftypes.String, "oops"),
			expectedErr: "can't unmarshal tftypes.String into *bool, expected boolean",
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			got, err := BoolType{}.ValueFromTerraform(ctx, test.input)
			if err != nil {
				if test.expectedErr == "" {
					t.Errorf("Unexpected error: %s", err)
					return
				}
				if test.expectedErr != err.Error() {
					t.Errorf("Expected error to be %q, got %q", test.expectedErr, err.Error())
					return
				}
				// we have an error, and it matches our
				// expectations, we're good
				return
			}
			if err == nil && test.expectedErr != "" {
				t.Errorf("Expected error to be %q, didn't get an error", test.expectedErr)
				return
			}
			if !got.Equal(test.expectation) {
				t.Errorf("Expected %+v, got %+v", test.expectation, got)
			}
			if test.expectation.IsNull() != test.input.IsNull() {
				t.Errorf("Expected null-ness match: expected %t, got %t", test.expectation.IsNull(), test.input.IsNull())
			}
			if test.expectation.IsUnknown() != !test.input.IsKnown() {
				t.Errorf("Expected unknown-ness match: expected %t, got %t", test.expectation.IsUnknown(), !test.input.IsKnown())
			}
		})
	}
}

func TestBoolValueToTerraformValue(t *testing.T) {
	t.Parallel()

	type testCase struct {
		input       BoolValue
		expectation interface{}
	}
	tests := map[string]testCase{
		"known-true": {
			input:       NewBoolValue(true),
			expectation: tftypes.NewValue(tftypes.Bool, true),
		},
		"known-false": {
			input:       NewBoolValue(false),
			expectation: tftypes.NewValue(tftypes.Bool, false),
		},
		"unknown": {
			input:       NewBoolUnknown(),
			expectation: tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		},
		"null": {
			input:       NewBoolNull(),
			expectation: tftypes.NewValue(tftypes.Bool, nil),
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			got, err := test.input.ToTerraformValue(ctx)
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
				return
			}
			if !cmp.Equal(got, test.expectation) {
				t.Errorf("Expected %+v, got %+v", test.expectation, got)
			}
		})
	}
}

func TestBoolValueEqual(t *testing.T) {
	t.Parallel()

	type testCase struct {
		input       BoolValue
		candidate   attr.Value
		expectation bool
	}
	tests := map[string]testCase{
		"known-true-nil": {
			input:       NewBoolValue(true),
			candidate:   nil,
			expectation: false,
		},
		"known-true-wrongtype": {
			input:       NewBoolValue(true),
			candidate:   NewStringValue("true"),
			expectation: false,
		},
		"known-true-known-false": {
			input:       NewBoolValue(true),
			candidate:   NewBoolValue(false),
			expectation: false,
		},
		"known-true-known-true": {
			input:       NewBoolValue(true),
			candidate:   NewBoolValue(true),
			expectation: true,
		},
		"known-true-null": {
			input:       NewBoolValue(true),
			candidate:   NewBoolNull(),
			expectation: false,
		},
		"known-true-unknown": {
			input:       NewBoolValue(true),
			candidate:   NewBoolUnknown(),
			expectation: false,
		},
		"known-false-nil": {
			input:       NewBoolValue(false),
			candidate:   nil,
			expectation: false,
		},
		"known-false-wrongtype": {
			input:       NewBoolValue(false),
			candidate:   NewStringValue("false"),
			expectation: false,
		},
		"known-false-known-false": {
			input:       NewBoolValue(false),
			candidate:   NewBoolValue(false),
			expectation: true,
		},
		"known-false-known-true": {
			input:       NewBoolValue(false),
			candidate:   NewBoolValue(true),
			expectation: false,
		},
		"known-false-null": {
			input:       NewBoolValue(false),
			candidate:   NewBoolNull(),
			expectation: false,
		},
		"known-false-unknown": {
			input:       NewBoolValue(false),
			candidate:   NewBoolUnknown(),
			expectation: false,
		},
		"null-nil": {
			input:       NewBoolNull(),
			candidate:   nil,
			expectation: false,
		},
		"null-wrongtype": {
			input:       NewBoolNull(),
			candidate:   NewStringValue("true"),
			expectation: false,
		},
		"null-known-false": {
			input:       NewBoolNull(),
			candidate:   NewBoolValue(false),
			expectation: false,
		},
		"null-known-true": {
			input:       NewBoolNull(),
			candidate:   NewBoolValue(true),
			expectation: false,
		},
		"null-null": {
			input:       NewBoolNull(),
			candidate:   NewBoolNull(),
			expectation: true,
		},
		"null-unknown": {
			input:       NewBoolNull(),
			candidate:   NewBoolUnknown(),
			expectation: false,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := test.input.Equal(test.candidate)
			if !cmp.Equal(got, test.expectation) {
				t.Errorf("Expected %v, got %v", test.expectation, got)
			}
		})
	}
}

func TestBoolValueIsNull(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    BoolValue
		expected bool
	}{
		"known": {
			input:    NewBoolValue(true),
			expected: false,
		},
		"null": {
			input:    NewBoolNull(),
			expected: true,
		},
		"unknown": {
			input:    NewBoolUnknown(),
			expected: false,
		},
	}

	for name, testCase := range testCases {
		name, testCase := name, testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := testCase.input.IsNull()

			if diff := cmp.Diff(got, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestBoolValueIsUnknown(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    BoolValue
		expected bool
	}{
		"known": {
			input:    NewBoolValue(true),
			expected: false,
		},
		"null": {
			input:    NewBoolNull(),
			expected: false,
		},
		"unknown": {
			input:    NewBoolUnknown(),
			expected: true,
		},
	}

	for name, testCase := range testCases {
		name, testCase := name, testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := testCase.input.IsUnknown()

			if diff := cmp.Diff(got, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestBoolValueString(t *testing.T) {
	t.Parallel()

	type testCase struct {
		input       BoolValue
		expectation string
	}
	tests := map[string]testCase{
		"known-true": {
			input:       NewBoolValue(true),
			expectation: "true",
		},
		"known-false": {
			input:       NewBoolValue(false),
			expectation: "false",
		},
		"null": {
			input:       NewBoolNull(),
			expectation: "<null>",
		},
		"unknown": {
			input:       NewBoolUnknown(),
			expectation: "<unknown>",
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := test.input.String()
			if !cmp.Equal(got, test.expectation) {
				t.Errorf("Expected %q, got %q", test.expectation, got)
			}
		})
	}
}

func TestBoolValueValueBool(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input    BoolValue
		expected bool
	}{
		"known-false": {
			input:    NewBoolValue(false),
			expected: false,
		},
		"known-true": {
			input:    NewBoolValue(true),
			expected: true,
		},
		"null": {
			input:    NewBoolNull(),
			expected: false,
		},
		"unknown": {
			input:    NewBoolUnknown(),
			expected: false,
		},
	}

	for name, testCase := range testCases {
		name, testCase := name, testCase

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := testCase.input.ValueBool()

			if diff := cmp.Diff(got, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}
