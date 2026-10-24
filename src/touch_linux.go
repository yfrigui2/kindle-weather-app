package main

import (
	"encoding/binary"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	evSyn = 0x00
	evKey = 0x01
	evAbs = 0x03

	absX          = 0x00
	absY          = 0x01
	absMTX        = 0x35
	absMTY        = 0x36
	absMTTracking = 0x39
	btnTouch      = 0x14a

	longPress = 3 * time.Second
)

type absInfo struct{ Value, Min, Max, Fuzz, Flat, Res int32 }

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

// EVIOCGABS(abs) = _IOR('E', 0x40 + abs, struct input_absinfo)
func eviocgabs(axis int) uintptr { return 2<<30 | 24<<16 | 'E'<<8 | uintptr(0x40+axis) }

// EVIOCGRAB = _IOW('E', 0x90, int)
const eviocgrab = 1<<30 | 4<<16 | 'E'<<8 | 0x90

type rawEvent struct {
	typ, code uint16
	value     int32
}

// watchTouch reports taps (on release) and long presses (after 3s held) in screen pixels.
func watchTouch(cfg *Config, out chan<- touch) {
	f, err := os.Open(cfg.TouchDevice)
	if err != nil {
		log.Printf("touch: %v (taps disabled)", err)
		return
	}
	fd := f.Fd()
	ranges := map[int]absInfo{}
	for _, a := range []int{absX, absY, absMTX, absMTY} {
		var ai absInfo
		if ioctl(fd, eviocgabs(a), unsafe.Pointer(&ai)) == nil && ai.Max > ai.Min {
			ranges[a] = ai
		}
	}
	log.Printf("touch ranges: %+v", ranges)
	// Take the touchscreen for ourselves while the display runs.
	one := int32(1)
	if err := ioctl(fd, eviocgrab, unsafe.Pointer(&one)); err != nil {
		log.Printf("touch grab: %v", err)
	}

	events := make(chan rawEvent, 64)
	go func() {
		buf := make([]byte, 16) // struct input_event on 32-bit ARM
		for {
			if _, err := f.Read(buf); err != nil {
				log.Printf("touch read: %v", err)
				close(events)
				return
			}
			events <- rawEvent{binary.LittleEndian.Uint16(buf[8:]), binary.LittleEndian.Uint16(buf[10:]), int32(binary.LittleEndian.Uint32(buf[12:]))}
		}
	}()

	scale := func(axis int, v int32, size int) int {
		ai, ok := ranges[axis]
		if !ok {
			return int(v)
		}
		return int(int64(v-ai.Min) * int64(size) / int64(ai.Max-ai.Min+1))
	}
	toScreen := func(rx, ry int32, xAxis, yAxis int) (int, int) {
		if cfg.TouchSwapXY {
			rx, ry = ry, rx
			xAxis, yAxis = yAxis, xAxis
		}
		x, y := scale(xAxis, rx, canvasW), scale(yAxis, ry, canvasH)
		if cfg.TouchFlipX {
			x = canvasW - 1 - x
		}
		if cfg.TouchFlipY {
			y = canvasH - 1 - y
		}
		return x, y
	}

	var (
		down, pending, fired bool
		rx, ry               int32
		xAxis, yAxis         = absX, absY
		startX, startY       int
		since                time.Time
		timer                <-chan time.Time
	)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			switch {
			case ev.typ == evAbs && (ev.code == absMTX || ev.code == absX):
				rx, xAxis = ev.value, int(ev.code)
			case ev.typ == evAbs && (ev.code == absMTY || ev.code == absY):
				ry, yAxis = ev.value, int(ev.code)
			case ev.typ == evKey && ev.code == btnTouch:
				if ev.value == 1 && !down {
					down, pending, fired = true, true, false
					since = time.Now()
					timer = time.After(longPress)
				} else if ev.value == 0 && down {
					down, timer = false, nil
					if !fired && time.Since(since) < 1500*time.Millisecond {
						out <- touch{x: startX, y: startY}
					}
				}
			case ev.typ == evAbs && ev.code == absMTTracking && ev.value != -1 && !down:
				down, pending, fired = true, true, false
				since = time.Now()
				timer = time.After(longPress)
			case ev.typ == evAbs && ev.code == absMTTracking && ev.value == -1 && down:
				down, timer = false, nil
				if !fired && time.Since(since) < 1500*time.Millisecond {
					out <- touch{x: startX, y: startY}
				}
			case ev.typ == evSyn && pending:
				// First full report after touch-down: where the finger landed.
				startX, startY = toScreen(rx, ry, xAxis, yAxis)
				pending = false
			}
		case <-timer:
			timer = nil
			if down {
				fired = true
				out <- touch{x: startX, y: startY, long: true}
			}
		}
	}
}
