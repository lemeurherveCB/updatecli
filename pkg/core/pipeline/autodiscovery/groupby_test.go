package autodiscovery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupByValidate(t *testing.T) {
	tests := []struct {
		name        string
		groupBy     GroupBy
		expectError bool
	}{
		{name: "empty is valid", groupBy: ""},
		{name: "all is valid", groupBy: GROUPBYALL},
		{name: "individual is valid", groupBy: GROUPBYINDIVIDUAL},
		{name: "resource is valid", groupBy: GROUPBYRESOURCE},
		{name: "unknown value is invalid", groupBy: "unknown", expectError: true},
		{
			// Regression test: Validate() used to compare len(g) == len(groupBy) instead of
			// g == groupBy, so any garbage value sharing a valid value's length (e.g. "all"
			// has 3 characters) was wrongly accepted.
			name:        "value matching a valid value's length only is invalid",
			groupBy:     "xyz",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.groupBy.Validate()
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
