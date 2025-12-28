package tracing

import "time"

// nowNanos returns the current time in nanoseconds since epoch.
func nowNanos() int64 {
	return time.Now().UnixNano()
}

// millisToTime converts milliseconds since epoch to time.Time.
func millisToTime(millis int64) time.Time {
	return time.Unix(0, millis*1_000_000)
}

// millisToDuration converts milliseconds to time.Duration.
func millisToDuration(millis int64) time.Duration {
	return time.Duration(millis) * time.Millisecond
}
