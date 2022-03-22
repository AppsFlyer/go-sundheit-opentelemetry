package sundheitotel

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    []attribute.KeyValue
		expected string
	}{
		{
			name:     "basic",
			input:    []attribute.KeyValue{attribute.String("a", "b"), attribute.String("c", "d")},
			expected: `[{"Key":"a","Value":{"Type":"STRING","Value":"b"}},{"Key":"c","Value":{"Type":"STRING","Value":"d"}}]`,
		},
		{
			name:  "sorting",
			input: []attribute.KeyValue{attribute.String("z", "3"), attribute.String("a", "1"), attribute.String("g", "2")},
			expected: `[{"Key":"a","Value":{"Type":"STRING","Value":"1"}},` +
				`{"Key":"g","Value":{"Type":"STRING","Value":"2"}},{"Key":"z","Value":{"Type":"STRING","Value":"3"}}]`,
		},
		{
			name: "types",
			input: []attribute.KeyValue{
				attribute.String("a", "1"),
				attribute.Bool("d", true),
			},
			expected: `[{"Key":"a","Value":{"Type":"STRING","Value":"1"}},{"Key":"d","Value":{"Type":"BOOL","Value":true}}]`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attributes, err := serializeAttributes(test.input)
			require.NoError(t, err, "unable to serialize attributes")
			require.Equal(t, test.expected, attributes)
			res, err := deserializeAttributes(attributes)
			require.NoError(t, err)
			require.Equal(t, test.input, res, "unable to deserialize attributes")
		})
	}
}
