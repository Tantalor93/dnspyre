package reporter

import "time"

func roundDuration(dur time.Duration) time.Duration {
	if dur > time.Minute {
		return dur.Round(10 * time.Second)
	}
	if dur > time.Second {
		return dur.Round(10 * time.Millisecond)
	}
	if dur > time.Millisecond {
		return dur.Round(10 * time.Microsecond)
	}
	if dur > time.Microsecond {
		return dur.Round(10 * time.Nanosecond)
	}
	return dur
}
