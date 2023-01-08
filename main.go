// Package leapseconds provides a means of determining when leap seconds are
// schedules as well as for those that have already occurred.  The code only
// provides a means of parsing the API endpoints provided by https://hpiers.obspm.fr/eop-pc/index.php
//
// This library does not provide an authoritive source of leap second
// information. The current authoritive source of leap second information is
// provided by the IERS in the data provided in Bulletin C (see
// https://www.iers.org/IERS/EN/Publications/Bulletins/bulletins.html).
package leapseconds // import "github.com/gwvandesteeg/leapseconds"

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	// ScheduledURL contains the default URI used for getting the next scheduled leap second data
	ScheduledURL string = "https://hpiers.obspm.fr/eop-pc/webservice/CURL/leapSecond.php"
	// DataSourceURL contains the URI to the data source containing all the leap seconds that have occurred
	DataSourceURL string = "https://hpiers.obspm.fr/iers/bul/bulc/Leap_Second.dat"
	notScheduled  string = "Not scheduled"
)

var (
	// ErrParsing indictes a parsing error of some sort
	ErrParsing = errors.New("parsing error")
	// ErrInvalidLastEntry indicates if an error parsing the scheduleds leap second data contains the incorret set of fields
	ErrInvalidScheduledEntry = errors.Wrap(ErrParsing, "scheduled leap second data")
	// ErrInvalidLeapSecondData indicates if an error occurred whilst parsing a line of leap second data where insufficient fields were found
	ErrInvalidLeapSecondData = errors.Wrap(ErrParsing, "wrong numbeer of leap second data points")
)

// LeapSeconds provides an interface on getting details about leap seconds
type LeapSeconds interface {
	// Get the next scheduled leap second (if one is announced)
	Scheduled() *time.Time
	// get the most recent leap second occurrence
	Last() time.Time
	// get a list of all leap seconds
	All() []time.Time
	// get the current TAI Offset (difference between astronomical time and UTC)
	TAIOffset() time.Duration
}

// Information provides a means of getting details about leap seconds
type Information struct {
	// Client provides a means of overriding the default HTTP client (which you should always do to set a sane set of timeouts)
	Client http.Client
	// EmbeddedFailover enables failover to the embedded data instead if the URI source is not available
	EmbeddedFailover bool

	scheduledURI  string
	scheduled     lastEntry
	historicalURI string
	historical    []leapSecondData
}

// check the struct implements the desired interface
var _ LeapSeconds = (*Information)(nil)

// Create a new instance of the information
func New() Information {
	return Information{
		Client:        *http.DefaultClient,
		historicalURI: DataSourceURL,
		scheduledURI:  ScheduledURL,
	}
}

func (i *Information) loadHistorical(ctx context.Context) error {
	var err error
	i.historical, err = fetchDataFileFromURL(ctx, i.historicalURI, i.Client)
	if err != nil {
		if !i.EmbeddedFailover {
			return err
		}
		i.historical, err = fetchDataFileFromEmbedded()
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Information) loadScheduled(ctx context.Context) error {
	got, err := fetchScheduledFromURL(ctx, i.scheduledURI, i.Client)
	if err != nil {
		return err
	}
	i.scheduled = *got
	return nil
}

func (i *Information) Load(ctx context.Context) error {
	if err := i.loadHistorical(ctx); err != nil {
		return err
	}
	if err := i.loadScheduled(ctx); err != nil {
		return err
	}
	return nil
}

func (i *Information) All() []time.Time {
	var rv []time.Time
	for _, e := range i.historical {
		rv = append(rv, e.Date)
	}
	return rv
}

func (i *Information) Last() time.Time {
	return i.scheduled.last
}

func (i *Information) Scheduled() *time.Time {
	return i.scheduled.next
}

func (i *Information) TAIOffset() time.Duration {
	return time.Duration(i.scheduled.taiOffset) * time.Second
}

type lastEntry struct {
	taiOffset uint16
	last      time.Time
	next      *time.Time
}

func newLastEntryFromString(line string) (*lastEntry, error) {
	var err error
	rv := &lastEntry{}
	items := strings.Split(line, "|")
	if len(items) != 3 {
		return nil, ErrInvalidScheduledEntry
	}
	rtai, err := strconv.ParseUint(items[0], 10, 16)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing TAI offset field containing %s", items[0])
	}
	rv.taiOffset = uint16(rtai)

	rv.last, err = time.Parse("02 January 2006", items[1])
	if err != nil {
		return nil, errors.Wrapf(err, "parsing last leap second date containing %s", items[1])
	}
	if items[2] != notScheduled {
		next, err := time.Parse("02 January 2006", items[2])
		if err != nil {
			return nil, errors.Wrapf(err, "parsing next scheduled leap second date containing %s", items[2])
		}
		rv.next = &next
	}

	return rv, nil
}

func parseScheduled(body io.Reader) (*lastEntry, error) {
	scanner := bufio.NewScanner(body)
	var line string
	if scanner.Scan() {
		line = scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return newLastEntryFromString(line)
}

func fetchScheduledFromURL(ctx context.Context, url string, client http.Client) (*lastEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return parseScheduled(res.Body)
}

type leapSecondData struct {
	mjd       float64
	day       uint8
	month     uint8
	year      uint16
	taiOffset uint16

	Date time.Time // the date of the leap second, the parsed leap second data is actually for the following day, not the day with the leap second
}

func (l *leapSecondData) updateDate() {
	l.Date = time.Date(int(l.year), time.Month(l.month), int(l.day), 0, 0, 0, 0, time.UTC).Add(-1 * time.Second)
}

func newLeapSecondDataFromString(line string) (*leapSecondData, error) {
	items := strings.Fields(line)
	if len(items) != 5 {
		return nil, ErrInvalidLeapSecondData
	}
	return newLeapSecondDataFromItems(items)
}

func newLeapSecondDataFromItems(items []string) (*leapSecondData, error) {
	var err error
	rv := &leapSecondData{}
	if rv.mjd, err = strconv.ParseFloat(items[0], 64); err != nil {
		return nil, errors.Wrapf(err, "parsing MJD field containing %s", items[0])
	}
	rday, err := strconv.ParseUint(items[1], 10, 8)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing day field containing %s", items[1])
	}
	rv.day = uint8(rday)
	rmonth, err := strconv.ParseUint(items[2], 10, 8)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing month field containing %s", items[2])
	}
	rv.month = uint8(rmonth)
	ryear, err := strconv.ParseUint(items[3], 10, 16)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing year field containing %s", items[3])
	}
	rv.year = uint16(ryear)
	rtai, err := strconv.ParseUint(items[4], 10, 16)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing TAI offset field containing %s", items[4])
	}
	rv.taiOffset = uint16(rtai)
	rv.updateDate()
	return rv, nil
}

// parseDataFile parses the io.Reader for leapsecond data
func parseDataFile(body io.Reader) ([]leapSecondData, error) {
	var rv []leapSecondData
	re := regexp.MustCompile("(?s)#.*?$")
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		// strip all comments from the line
		line := re.ReplaceAll(scanner.Bytes(), nil)
		// split the line into words
		items := strings.Fields(string(line))
		if len(items) > 0 {
			lsd, err := newLeapSecondDataFromItems(items)
			if err != nil {
				return rv, err
			}
			rv = append(rv, *lsd)
		}
	}
	if err := scanner.Err(); err != nil {
		return rv, err
	}
	return rv, nil
}

// fetchDataFileFromURL retrieves a data file from the provided URL
func fetchDataFileFromURL(ctx context.Context, url string, client http.Client) ([]leapSecondData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return fetchDataFileFromReader(res.Body)
}

//go:embed assets/Leap_Second.dat
var embeddedLeapSecondData []byte

// fetchDataFileFromEmbedded retrieves a data file from the build time embedded data asset
func fetchDataFileFromEmbedded() ([]leapSecondData, error) {
	b := bytes.NewReader(embeddedLeapSecondData)
	return fetchDataFileFromReader(b)
}

// fetchDataFileFromReader retrieves a data file from io.Reader
func fetchDataFileFromReader(src io.Reader) ([]leapSecondData, error) {
	parsed, err := parseDataFile(src)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}
