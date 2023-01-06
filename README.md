# leapseconds

Go library to provide information about historical and scheduled leap seconds.

Leap seconds are applied to UTC, to accomodate the difference between the precise time (International Atomic Time (TAI)) as measured by atomic clocks, and the imprecise observed solar time (UT1) which varies due to irregularities and long-term slowdown in the Earth's rotation.

The code only provides a means of parsing the API endpoints provided by the [L’Observatoire de Paris](https://hpiers.obspm.fr/eop-pc/index.php) which in turn retrieves its information from the [International Earth Rotation and Reference Systems Service](https://iers.org) as published in Bulletin C. Up and coming leap seconds should be announced by the IERS up to six months in advance, however this does not guarantee that the API this library used is updated at the same time.

Warning: There has been a debate since 2005 on whether leap seconds should be eliminated and has not drawn to a conclusion.

`go get github.com/gwvandesteeg/leapseconds`

## Create an information source

```go
info := leapseconds.New()
info.Client := http.Client{Timeout: 5*time.Second}
if err := info.Load(context.TODO()); err != nil {
  panic(err)
}

fmt.Printf("last leapsecond occurred at: %v", info.Last())
```
