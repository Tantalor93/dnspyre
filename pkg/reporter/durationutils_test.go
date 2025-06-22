package reporter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_roundDuration(t *testing.T) {
	type args struct {
		dur time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "greater than 1 minute",
			args: args{dur: time.Minute + 15*time.Second + 123*time.Millisecond},
			want: 1*time.Minute + 20*time.Second,
		},
		{
			name: "greater than 1 second",
			args: args{dur: 5*time.Second + 123*time.Millisecond},
			want: 5*time.Second + 120*time.Millisecond,
		},
		{
			name: "greater than 1 millisecond",
			args: args{dur: 2*time.Millisecond + 123*time.Microsecond},
			want: 2*time.Millisecond + 120*time.Microsecond,
		},
		{
			name: "greater than 1 microsecond",
			args: args{dur: 2*time.Microsecond + 123*time.Nanosecond},
			want: 2*time.Microsecond + 120*time.Nanosecond,
		},
		{
			name: "less than or equal to 1 microsecond",
			args: args{dur: 500 * time.Nanosecond},
			want: 500 * time.Nanosecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, roundDuration(tt.args.dur), "roundDuration(%v)", tt.args.dur)
		})
	}
}
