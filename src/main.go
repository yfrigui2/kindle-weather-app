// pocketink draws the Pocket Ink weather screens on a Kindle's e-ink panel.
//
//	pocketink run -config pocketink.json      the always-on display (on the Kindle)
//	pocketink render -sample -screen main     render a PNG (for checking on a computer)
//	pocketink touchprobe                      print touch events, to calibrate taps
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: pocketink run|render|touchprobe [flags]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		cmdRun(os.Args[2:])
	case "render":
		cmdRender(os.Args[2:])
	case "touchprobe":
		cmdTouchProbe(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, "unknown command", os.Args[1])
		os.Exit(2)
	}
}

func loadConfig(path string) *Config {
	cfg := &Config{Timezone: "auto", Units: "imperial", RefreshMinutes: 15, Theme: "paper", TouchDevice: "/dev/input/event2"}
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		log.Fatalf("config %s: %v", path, err)
	}
	if cfg.RefreshMinutes < 5 {
		cfg.RefreshMinutes = 5
	}
	return cfg
}

func writePNG(img image.Image, path string) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func cmdRender(args []string) {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	sample := fs.Bool("sample", false, "use the design system's preview data")
	data := fs.String("data", "", "a saved Open-Meteo response")
	at := fs.String("at", "", "fetch time for -data (RFC 3339), default now")
	screen := fs.String("screen", "main", "main, days, offline or bye")
	night := fs.Bool("night", false, "Night theme")
	out := fs.String("o", "screen.png", "output PNG")
	fs.Parse(args)

	var m *Model
	switch {
	case *sample:
		m = sampleModel()
	case *data != "":
		raw, err := os.ReadFile(*data)
		if err != nil {
			log.Fatal(err)
		}
		t := time.Now()
		if *at != "" {
			if t, err = time.Parse(time.RFC3339, *at); err != nil {
				log.Fatal(err)
			}
		}
		if m, err = buildModel(raw, t, 60, &Config{}); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("render needs -sample or -data")
	}
	var img *image.Gray
	switch *screen {
	case "days":
		img = renderDays(m, *night)
	case "offline":
		img = renderMessage(m, "OFFLINE.EXE", "storm", "can't reach the weather yet. check the wi-fi; trying again soon.", true, *night)
	case "bye":
		img = renderMessage(m, "BYE.EXE", "sparkle", "back to the kindle in a minute or two.", false, *night)
	default:
		img = renderMain(m, *night)
	}
	if err := writePNG(img, *out); err != nil {
		log.Fatal(err)
	}
}

// ---------- the always-on display ----------

type display struct {
	cfg     *Config
	dir     string
	raw     []byte
	fetched time.Time
	screen  string
}

func (d *display) cachePath() string { return filepath.Join(d.dir, "cache.json") }

func batteryLevel() int {
	out, err := exec.Command("lipc-get-prop", "com.lab126.powerd", "battLevel").Output()
	if err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			return v
		}
	}
	return -1
}

// ensureWifi brings Wi-Fi back if it dropped. With the Kindle UI paused nothing else
// rejoins the network once it returns (after a router restart, say), so turn the radio
// off and on: the Kindle then reconnects to its saved networks.
func ensureWifi() {
	state := func() string {
		out, _ := exec.Command("lipc-get-prop", "com.lab126.wifid", "cmState").Output()
		return strings.TrimSpace(string(out))
	}
	s := state()
	if s == "CONNECTED" {
		return
	}
	log.Printf("wifi is %q, restarting it", s)
	exec.Command("lipc-set-prop", "com.lab126.cmd", "wirelessEnable", "0").Run()
	time.Sleep(5 * time.Second)
	exec.Command("lipc-set-prop", "com.lab126.cmd", "wirelessEnable", "1").Run()
	for i := 0; i < 45 && state() != "CONNECTED"; i++ {
		time.Sleep(2 * time.Second)
	}
	log.Printf("wifi is %q", state())
}

type fetchResult struct {
	raw []byte
	at  time.Time
	err error
}

// fetchWeather tries a few times, reconnecting Wi-Fi as needed. It runs in the
// background so taps keep working while it waits.
func fetchWeather(cfg *Config) fetchResult {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		ensureWifi()
		var raw []byte
		if raw, err = fetch(cfg); err == nil {
			return fetchResult{raw: raw, at: time.Now()}
		}
		log.Printf("fetch attempt %d: %v", attempt, err)
		time.Sleep(20 * time.Second)
	}
	return fetchResult{err: err}
}

func (d *display) saveCache() {
	cache, _ := json.Marshal(map[string]any{"fetched": d.fetched.Format(time.RFC3339), "data": json.RawMessage(d.raw)})
	if err := os.WriteFile(d.cachePath(), cache, 0o644); err != nil {
		log.Printf("cache: %v", err)
	}
}

func (d *display) loadCache() {
	b, err := os.ReadFile(d.cachePath())
	if err != nil {
		return
	}
	var c struct {
		Fetched string          `json:"fetched"`
		Data    json.RawMessage `json:"data"`
	}
	if json.Unmarshal(b, &c) != nil {
		return
	}
	if t, err := time.Parse(time.RFC3339, c.Fetched); err == nil {
		d.raw, d.fetched = c.Data, t
	}
}

// staleAfter is how old a forecast may get before the screen stops presenting it as now.
const staleAfter = 3 * time.Hour

// updatedLabel is "UPDATED 6:00 AM", with the weekday once it is no longer today.
func updatedLabel(fetched, now time.Time) string {
	hm, ap := clock(fetched)
	if fetched.Format("2006-01-02") != now.Format("2006-01-02") {
		return "UPDATED " + strings.ToUpper(fetched.Format("Mon")) + " " + hm + " " + ap
	}
	return "UPDATED " + hm + " " + ap
}

func (d *display) show() {
	night := d.cfg.Theme == "night"
	now := time.Now()
	var img *image.Gray
	m, err := (*Model)(nil), error(nil)
	if d.raw != nil {
		m, err = buildModel(d.raw, d.fetched, batteryLevel(), d.cfg)
		if err != nil {
			log.Printf("model: %v", err)
		}
	}
	if m != nil {
		// The date is today's; the forecast says when it was fetched.
		loc := time.FixedZone("local", tzOffset(d.raw))
		m.DateLabel = dateLabel(now.In(loc))
		m.Updated = updatedLabel(d.fetched.In(loc), now.In(loc))
	}
	switch {
	case m == nil:
		hm, ap := clock(now)
		blank := &Model{DateLabel: dateLabel(now), Updated: "TRIED " + hm + " " + ap, Battery: batteryLevel()}
		img = renderMessage(blank, "OFFLINE.EXE", "storm", "can't reach the weather yet. check the wi-fi; trying again soon.", true, night)
	case now.Sub(d.fetched) > staleAfter:
		since := strings.ToLower(strings.TrimPrefix(m.Updated, "UPDATED "))
		msg := fmt.Sprintf("no weather since %s. check the wi-fi; trying again every %d minutes.", since, d.cfg.RefreshMinutes)
		img = renderMessage(m, "OFFLINE.EXE", "storm", msg, true, night)
	case d.screen == "days":
		img = renderDays(m, night)
	default:
		img = renderMain(m, night)
	}
	d.paint(img)
}

// goodbye stays on screen while the Kindle UI restarts, which takes a minute or two.
func (d *display) goodbye() {
	now := time.Now()
	m := &Model{DateLabel: dateLabel(now), Updated: "", Battery: batteryLevel()}
	d.paint(renderMessage(m, "BYE.EXE", "sparkle", "back to the kindle in a minute or two.", false, d.cfg.Theme == "night"))
}

func (d *display) paint(img *image.Gray) {
	path := filepath.Join(d.dir, "screen.png")
	if err := writePNG(img, path); err != nil {
		log.Printf("write: %v", err)
		return
	}
	// Full refresh every time, so the last screen doesn't ghost through.
	if out, err := exec.Command("eips", "-f", "-g", path).CombinedOutput(); err != nil {
		log.Printf("eips: %v: %s", err, out)
	}
}

func nextSlot(now time.Time, minutes int) time.Time {
	step := time.Duration(minutes) * time.Minute
	return now.Truncate(step).Add(step)
}

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	cfgPath := fs.String("config", "pocketink.json", "config file")
	fs.Parse(args)
	cfg := loadConfig(*cfgPath)
	d := &display{cfg: cfg, dir: filepath.Dir(*cfgPath), screen: "main"}
	d.loadCache()
	log.Printf("starting: %s (%.4f, %.4f), every %d min", cfg.Name, cfg.Latitude, cfg.Longitude, cfg.RefreshMinutes)

	touches := make(chan touch, 8)
	go watchTouch(cfg, touches)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	// Show the last forecast right away; the first update replaces it.
	if d.raw != nil {
		d.show()
	}
	results := make(chan fetchResult, 1)
	fetching := false
	next := time.Now()
	for {
		if !fetching && !time.Now().Before(next) {
			fetching = true
			go func() { results <- fetchWeather(cfg) }()
			next = nextSlot(time.Now(), cfg.RefreshMinutes)
		}
		select {
		case r := <-results:
			fetching = false
			if r.err == nil {
				d.raw, d.fetched = r.raw, r.at
				d.saveCache()
				log.Printf("fetched weather")
			}
			d.show() // redraw whichever screen is showing with the new forecast
		case t := <-touches:
			p := image.Pt(t.x, t.y)
			if t.long {
				// Only a hold on the status bar exits, so picking the Kindle up doesn't.
				if p.In(hitStatusBar) {
					log.Printf("long press on the status bar: exiting")
					d.goodbye()
					return
				}
				continue
			}
			log.Printf("tap at %v on the %s screen", p, d.screen)
			switch {
			case d.screen == "main" && p.In(hitDaysKey):
				d.screen = "days"
				d.show()
			case d.screen == "days" && p.In(hitTodayKey):
				d.screen = "main"
				d.show()
			}
		case sig := <-stop:
			log.Printf("%v: exiting", sig)
			d.goodbye()
			return
		case <-wait(fetching, next):
		}
	}
}

// wait fires at the next update slot; while a fetch is running its result wakes the loop.
func wait(fetching bool, next time.Time) <-chan time.Time {
	if fetching {
		return nil
	}
	return time.After(time.Until(next))
}

func cmdTouchProbe(args []string) {
	fs := flag.NewFlagSet("touchprobe", flag.ExitOnError)
	cfgPath := fs.String("config", "pocketink.json", "config file")
	secs := fs.Int("seconds", 30, "how long to listen")
	fs.Parse(args)
	cfg := loadConfig(*cfgPath)
	touches := make(chan touch, 8)
	go watchTouch(cfg, touches)
	timeout := time.After(time.Duration(*secs) * time.Second)
	for {
		select {
		case t := <-touches:
			fmt.Printf("touch x=%d y=%d long=%v\n", t.x, t.y, t.long)
		case <-timeout:
			return
		}
	}
}
