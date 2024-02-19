package functions

import "testing"

func Test_Float(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected float64
	}{
		{
			name:     "Float with 1",
			args:     []interface{}{1},
			expected: 1.0,
		},
		{
			name:     "Float with 2",
			args:     []interface{}{"2"},
			expected: 2.0,
		},
		{
			name:     "Float with 3",
			args:     []interface{}{"2.3"},
			expected: 2.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Float(tt.args...)
			if got != tt.expected {
				t.Errorf("Float() = %v, want %v", got, tt.expected)
			}
		})
	}
}
