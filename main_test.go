package leapseconds

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"
)

func Example() {
	info := New()
	info.Client = http.Client{Timeout: 5 * time.Second}
	if err := info.Load(context.TODO()); err != nil {
		panic(err)
	}
	fmt.Printf("last leapsecond occurred at: %v\n", info.Last())
	// Output:
	// last leapsecond occurred at: 2016-12-31 00:00:00 +0000 UTC
}

func Test_fetchDataFileFromURL(t *testing.T) {
	_, err := fetchDataFileFromURL(context.TODO(), DataSourceURL, *http.DefaultClient)
	if err != nil {
		t.Errorf("fetchDataFileFromURL() error = %v", err)
	}
}

func Test_fetchScheduledFromURL(t *testing.T) {
	_, err := fetchScheduledFromURL(context.TODO(), ScheduledURL, *http.DefaultClient)
	if err != nil {
		t.Errorf("fetchScheduledFromURLL() error = %v", err)
	}
}

func BenchmarkParseDataFile(b *testing.B) {
	content, err := os.ReadFile("assets/Leap_Second.dat")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(content)
		if _, err := parseDataFile(r); err != nil {
			b.Fatal(err)
		}
	}
}

func Test_newLastEntryFromString(t *testing.T) {
	t1 := time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2015, 6, 30, 0, 0, 0, 0, time.UTC)
	type args struct {
		line string
	}
	tests := []struct {
		name    string
		args    args
		want    *lastEntry
		wantErr bool
	}{
		{"not scheduled",
			args{"37|31 December 2016|Not scheduled"},
			&lastEntry{
				taiOffset: 37,
				last:      t1,
				next:      nil,
			},
			false},
		{"next scheduled",
			args{"36|30 June 2015|31 December 2016"},
			&lastEntry{
				taiOffset: 36,
				last:      t2,
				next:      &t1,
			},
			false},
		{"invalid empty string", args{""}, nil, true},
		{"invalid insufficient fields", args{"10|31 December 2016"}, nil, true},
		{"invalid too many fields", args{"10|31 December 2016|Not scheduled|foo"}, nil, true},
		{"invalid TAI Offset string", args{"foo|31 December 2016|Not scheduled"}, nil, true},
		{"invalid TAI Offset -ve value", args{"-10|31 December 2016|Not scheduled"}, nil, true},
		{"invalid last date", args{"10|foo|Not scheduled"}, nil, true},
		{"invalid scheduled date", args{"10|31 December 2016|foo"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newLastEntryFromString(tt.args.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("newLastEntryFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newLastEntryFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newLeapSecondDataFromString(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name    string
		args    args
		want    *leapSecondData
		wantErr bool
	}{
		{"valid", args{"41317.0    1  1 1972       10"}, &leapSecondData{
			mjd:       41317.0,
			day:       1,
			month:     1,
			year:      1972,
			taiOffset: 10,
			Date:      time.Date(1971, 12, 31, 23, 59, 59, 0, time.UTC),
		}, false},
		{"invalid empty string", args{""}, nil, true},
		{"invalid insufficient fields", args{"0.0 1 2 1903"}, nil, true},
		{"invalid too many fields", args{"0.0 1 2 1903 4 5 6"}, nil, true},
		{"invalid MJD", args{"foo     1  1 1972       10"}, nil, true},
		{"invalid day", args{"10.0     foo  1 1972       10"}, nil, true},
		{"invalid month", args{"10.0     1  foo 1972       10"}, nil, true},
		{"invalid month", args{"10.0     1  1 foo       10"}, nil, true},
		{"invalid TAI Offset", args{"10.0     1  1 1972       foo"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newLeapSecondDataFromString(tt.args.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("newLeapSecondDataFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newLeapSecondDataFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
