package repository

import (
	"errors"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestIsUndefinedTableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "postgres code", err: &pq.Error{Code: "42P01"}, want: true},
		{name: "other postgres code", err: &pq.Error{Code: "23505"}, want: false},
		{name: "fallback message", err: errors.New(`pq: relation "scheduler_history" does not exist`), want: true},
		{name: "unrelated error", err: errors.New("connection reset"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isUndefinedTableError(tt.err))
		})
	}
}
