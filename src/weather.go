package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Name           string  `json:"name"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Timezone       string  `json:"timezone"`
	Units          string  `json:"units"` // "imperial" (°F, mph) or "metric" (°C, km/h)
	RefreshMinutes int     `json:"refresh_minutes"`
	Theme          string  `json:"theme"` // "paper" or "night"
	TouchDevice    string  `json:"touch_device"`
	TouchSwapXY    bool    `json:"touch_swap_xy"`
	TouchFlipX     bool    `json:"touch_flip_x"`
	TouchFlipY     bool    `json:"touch_flip_y"`
}

type Hour struct {
	Label, Icon, Temp string
	Rain              int
}

type Day struct {
	Label, Icon, Word, Hi, Lo string
	Rain                      int
}

// Model is everything one screen shows, already formatted.
type Model struct {
	DateLabel, Updated                  string
	Battery                             int // 0-100, or -1 to hide the hearts
	Temp, Feels, Word, Icon, High       string
	SunIcon, SunLabel, SunTime, SunUnit string
	Hours                               []Hour
	Days                                []Day
}

func forecastURL(cfg *Config) string {
	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(cfg.Latitude, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(cfg.Longitude, 'f', 4, 64))
	q.Set("current", "temperature_2m,apparent_temperature,weather_code,is_day,wind_speed_10m")
	q.Set("hourly", "temperature_2m,precipitation_probability,weather_code,is_day")
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,sunrise,sunset,wind_speed_10m_max")
	q.Set("timezone", cfg.Timezone)
	q.Set("forecast_days", "7")
	if cfg.Units != "metric" {
		q.Set("temperature_unit", "fahrenheit")
		q.Set("wind_speed_unit", "mph")
	}
	return "https://api.open-meteo.com/v1/forecast?" + q.Encode()
}

// fetch uses the Kindle's own curl, which already trusts the system certificates.
func fetch(cfg *Config) ([]byte, error) {
	out, err := exec.Command("curl", "-sS", "--fail", "-m", "30", forecastURL(cfg)).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("curl: %v: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	var probe omResponse
	if err := json.Unmarshal(out, &probe); err != nil || len(probe.Daily.Time) < 5 {
		return nil, fmt.Errorf("unexpected response: %.200s", out)
	}
	return out, nil
}

type omResponse struct {
	UTCOffset int `json:"utc_offset_seconds"`
	Current   struct {
		Temp  *float64 `json:"temperature_2m"`
		Feels *float64 `json:"apparent_temperature"`
		Code  *float64 `json:"weather_code"`
		IsDay *float64 `json:"is_day"`
		Wind  *float64 `json:"wind_speed_10m"`
	} `json:"current"`
	Hourly struct {
		Time  []string   `json:"time"`
		Temp  []*float64 `json:"temperature_2m"`
		Pop   []*float64 `json:"precipitation_probability"`
		Code  []*float64 `json:"weather_code"`
		IsDay []*float64 `json:"is_day"`
	} `json:"hourly"`
	Daily struct {
		Time    []string   `json:"time"`
		Code    []*float64 `json:"weather_code"`
		Max     []*float64 `json:"temperature_2m_max"`
		Min     []*float64 `json:"temperature_2m_min"`
		Pop     []*float64 `json:"precipitation_probability_max"`
		Sunrise []string   `json:"sunrise"`
		Sunset  []string   `json:"sunset"`
		Wind    []*float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

func val(p *float64) float64 {
	if p == nil {
		return math.NaN()
	}
	return *p
}

func at(a []*float64, i int) float64 {
	if i < 0 || i >= len(a) {
		return math.NaN()
	}
	return val(a[i])
}

func deg(v float64) string {
	if math.IsNaN(v) {
		return "--"
	}
	r := int(math.Round(v))
	if r == 0 {
		return "0" // no "-0"
	}
	return itoa(r)
}

func pct(v float64) int {
	if math.IsNaN(v) {
		return -1
	}
	return int(math.Round(v))
}

func itoa(i int) string { return strconv.Itoa(i) }

// condition maps a WMO weather code to the design system's icon and voice.
// Exclamation marks only for nice weather.
func condition(code float64, day bool, wind, windyAt float64) (icon, word string) {
	c := int(code)
	if math.IsNaN(code) {
		return "cloud", "no data"
	}
	if c <= 3 && wind >= windyAt {
		return "wind", "windy"
	}
	switch c {
	case 0, 1:
		if day {
			return "sun", "sunny!"
		}
		return "moon", "clear"
	case 2:
		if day {
			return "partly", "some sun"
		}
		return "moon", "some clouds"
	case 3:
		return "cloud", "cloudy"
	case 45, 48:
		return "fog", "foggy"
	case 51, 53, 55:
		return "rain", "drizzle"
	case 56, 57, 66, 67:
		return "snow", "sleet"
	case 61, 63:
		return "rain", "rain"
	case 65:
		return "rain", "heavy rain"
	case 80, 81:
		return "rain", "showers"
	case 82:
		return "rain", "downpour"
	case 71, 73, 77, 85:
		return "snow", "snow"
	case 75, 86:
		return "snow", "snow day!"
	case 95, 96, 99:
		return "storm", "storms"
	}
	return "cloud", "cloudy"
}

func clock(t time.Time) (string, string) {
	return t.Format("3:04"), t.Format("PM")
}

func dateLabel(t time.Time) string {
	return strings.ToUpper(t.Format("Mon") + " · " + t.Format("Jan 2"))
}

// buildModel turns a saved Open-Meteo response into the screen model, as of when it was fetched.
func buildModel(raw []byte, fetched time.Time, battery int, cfg *Config) (*Model, error) {
	var r omResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	loc := time.FixedZone("local", r.UTCOffset)
	now := fetched.In(loc)
	parse := func(s string) time.Time {
		layout := "2006-01-02T15:04"
		if len(s) == 10 {
			layout = "2006-01-02"
		}
		t, _ := time.ParseInLocation(layout, s, loc)
		return t
	}
	windyAt := 25.0
	if cfg.Units == "metric" {
		windyAt = 40
	}

	m := &Model{Battery: battery, DateLabel: dateLabel(now)}
	hm, ap := clock(now)
	m.Updated = "UPDATED " + hm + " " + ap

	m.Temp = deg(val(r.Current.Temp))
	m.Feels = deg(val(r.Current.Feels))
	m.Icon, m.Word = condition(val(r.Current.Code), val(r.Current.IsDay) != 0, val(r.Current.Wind), windyAt)

	// Today is the daily entry for the local date.
	today := 0
	for i, d := range r.Daily.Time {
		if d == now.Format("2006-01-02") {
			today = i
		}
	}
	m.High = deg(at(r.Daily.Max, today))

	// Whichever sun event comes next.
	sunrise := func(i int) time.Time {
		if i < len(r.Daily.Sunrise) {
			return parse(r.Daily.Sunrise[i])
		}
		return time.Time{}
	}
	sunset := func(i int) time.Time {
		if i < len(r.Daily.Sunset) {
			return parse(r.Daily.Sunset[i])
		}
		return time.Time{}
	}
	var ev time.Time
	switch {
	case now.Before(sunrise(today)):
		ev, m.SunIcon, m.SunLabel = sunrise(today), "sunrise", "SUNRISE"
	case now.Before(sunset(today)):
		ev, m.SunIcon, m.SunLabel = sunset(today), "sunset", "SUNSET"
	default:
		ev, m.SunIcon, m.SunLabel = sunrise(today+1), "sunrise", "SUNRISE"
	}
	m.SunTime, m.SunUnit = clock(ev)

	// The next five whole hours.
	for i, ts := range r.Hourly.Time {
		if len(m.Hours) == 5 {
			break
		}
		t := parse(ts)
		if !t.After(now) {
			continue
		}
		icon, _ := condition(at(r.Hourly.Code, i), at(r.Hourly.IsDay, i) != 0, 0, windyAt)
		m.Hours = append(m.Hours, Hour{
			Label: strings.ToUpper(t.Format("3PM")),
			Icon:  icon,
			Temp:  deg(at(r.Hourly.Temp, i)),
			Rain:  pct(at(r.Hourly.Pop, i)),
		})
	}

	// Five days starting today. The days list drops the exclamation marks.
	for i := today; i < len(r.Daily.Time) && len(m.Days) < 5; i++ {
		icon, word := condition(at(r.Daily.Code, i), true, at(r.Daily.Wind, i), windyAt)
		m.Days = append(m.Days, Day{
			Label: strings.ToUpper(parse(r.Daily.Time[i]).Format("Mon")),
			Icon:  icon,
			Word:  strings.TrimSuffix(word, "!"),
			Hi:    deg(at(r.Daily.Max, i)),
			Lo:    deg(at(r.Daily.Min, i)),
			Rain:  pct(at(r.Daily.Pop, i)),
		})
	}
	if len(m.Hours) < 5 || len(m.Days) < 5 {
		return nil, fmt.Errorf("forecast too short: %d hours, %d days", len(m.Hours), len(m.Days))
	}
	return m, nil
}

// sampleModel reproduces the design system's WeatherScreen and DaysScreen previews.
func sampleModel() *Model {
	return &Model{
		DateLabel: "FRI · OCT 23", Updated: "UPDATED 5:40 PM", Battery: 60,
		Temp: "72", Feels: "70", Word: "sunny!", Icon: "sun", High: "74",
		SunIcon: "sunset", SunLabel: "SUNSET", SunTime: "6:42", SunUnit: "PM",
		Hours: []Hour{
			{"6PM", "sun", "71", -1}, {"7PM", "moon", "68", -1}, {"8PM", "cloud", "66", 30},
			{"9PM", "rain", "63", 70}, {"10PM", "rain", "61", 80},
		},
		Days: []Day{
			{"FRI", "sun", "sunny", "74", "58", -1}, {"SAT", "rain", "rain", "64", "52", 80},
			{"SUN", "storm", "storms", "59", "47", 90}, {"MON", "partly", "some sun", "62", "45", 20},
			{"TUE", "cloud", "cloudy", "60", "44", -1},
		},
	}
}
