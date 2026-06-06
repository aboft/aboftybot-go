package utils

import "testing"

func TestLineCountKey(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		date    string
		want    string
	}{
		{
			name:    "channel and date",
			channel: "#chan",
			date:    "2024-01-01",
			want:    "#chan|2024-01-01",
		},
		{
			name:    "empty date",
			channel: "#chan",
			date:    "",
			want:    "#chan|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lineCountKey(tt.channel, tt.date)
			if got != tt.want {
				t.Errorf("lineCountKey(%q, %q) = %q, want %q", tt.channel, tt.date, got, tt.want)
			}
		})
	}
}

func TestGetLineCountInvalidDate(t *testing.T) {
	// An unparseable date returns before the DB client is ever used,
	// so it is safe to pass a nil client here.
	got := GetLineCount(nil, "#chan", "nope")
	want := "Unable to parse date provided; use YYYY-MM-DD"
	if got != want {
		t.Errorf("GetLineCount(nil, %q, %q) = %q, want %q", "#chan", "nope", got, want)
	}
}

func TestGetLastNDaysLineCountsInvalidDays(t *testing.T) {
	tests := []struct {
		name string
		days string
		want string
	}{
		{
			name: "not a number",
			days: "abc",
			want: "Invalid number of days: abc",
		},
		{
			name: "zero",
			days: "0",
			want: "Invalid number of days: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Invalid days return before the DB client is used, so nil is safe.
			got := GetLastNDaysLineCounts(nil, "#chan", tt.days)
			if got != tt.want {
				t.Errorf("GetLastNDaysLineCounts(nil, %q, %q) = %q, want %q", "#chan", tt.days, got, tt.want)
			}
		})
	}
}
