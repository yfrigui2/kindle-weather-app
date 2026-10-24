package main

import (
	"embed"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Canvas and palette: tokens.json, Paper theme. Night is the whole screen inverted.
const (
	canvasW = 600
	canvasH = 800

	paper uint8 = 0xff
	grid  uint8 = 0xdd
	mist  uint8 = 0xee
	fog   uint8 = 0xbb
	slate uint8 = 0x55
	ink   uint8 = 0x00
)

//go:embed fonts/*.ttf
var fontFS embed.FS

type family int

const (
	fDigits    family = iota // Press Start 2P
	fPixel                   // Pixelify Sans 400
	fPixelSemi               // Pixelify Sans 600
	fLabel                   // Silkscreen 400
	fLabelBold               // Silkscreen 700
)

var fontFiles = map[family]string{
	fDigits:    "fonts/PressStart2P-Regular.ttf",
	fPixel:     "fonts/PixelifySans-Regular.ttf",
	fPixelSemi: "fonts/PixelifySans-SemiBold.ttf",
	fLabel:     "fonts/Silkscreen-Regular.ttf",
	fLabelBold: "fonts/Silkscreen-Bold.ttf",
}

var (
	fonts = map[family]*opentype.Font{}
	faces = map[style]font.Face{}
)

// A text style: font, size, CSS line-height and letter-spacing, all in px.
type style struct {
	fam     family
	size    float64
	lh      float64
	spacing float64
}

var (
	stLabel     = style{fLabel, 16, 16, 0.64}  // label: window titles, captions, status bar
	stColLabel  = style{fLabelBold, 24, 24, 0} // label-lg: forecast column headers, keys
	stNumHero   = style{fDigits, 112, 112, 0}  // num-hero
	stNumHeroXL = style{fDigits, 96, 112, 0}   // num-hero for four characters (-12°)
	stNumMd     = style{fDigits, 32, 32, 0}    // num-md
	stNumLong   = style{fDigits, 24, 32, 0}    // num-md value of four characters
	stNumSmPlus = style{fDigits, 24, 24, 0}    // num-sm-plus
	stNumSm     = style{fDigits, 16, 16, 0}    // num-sm
	stDisplay   = style{fPixelSemi, 56, 60, 0} // display
	stWord      = style{fPixel, 24, 28, 0}     // forecast word
	stBody      = style{fPixel, 24, 32, 0}     // body
)

func loadFont(f family) *opentype.Font {
	if ft, ok := fonts[f]; ok {
		return ft
	}
	b, err := fontFS.ReadFile(fontFiles[f])
	if err != nil {
		panic(err)
	}
	ft, err := opentype.Parse(b)
	if err != nil {
		panic(err)
	}
	fonts[f] = ft
	return ft
}

func faceFor(st style) font.Face {
	key := style{fam: st.fam, size: st.size}
	if fc, ok := faces[key]; ok {
		return fc
	}
	fc, err := opentype.NewFace(loadFont(st.fam), &opentype.FaceOptions{Size: st.size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	faces[key] = fc
	return fc
}

type canvas struct{ img *image.Gray }

func newCanvas() *canvas {
	return &canvas{image.NewGray(image.Rect(0, 0, canvasW, canvasH))}
}

func (c *canvas) fill(x0, y0, x1, y1 int, v uint8) {
	draw.Draw(c.img, image.Rect(x0, y0, x1, y1), image.NewUniform(color.Gray{v}), image.Point{}, draw.Src)
}

// Graph paper: 2px grid lines every grid-cell (24px), offset -1px like the CSS.
func (c *canvas) graphPaper() {
	c.fill(0, 0, canvasW, canvasH, paper)
	for k := 0; k <= canvasW/24+1; k++ {
		c.fill(24*k-1, 0, 24*k+1, canvasH, grid)
	}
	for k := 0; k <= canvasH/24+1; k++ {
		c.fill(0, 24*k-1, canvasW, 24*k+1, grid)
	}
}

// lineBox returns how far the baseline sits below the top of a line box, and the box height.
func lineBox(st style) (above, height int) {
	m := faceFor(st).Metrics()
	a := float64(m.Ascent.Round())
	d := float64(m.Descent.Round())
	return int(math.Floor(a + (st.lh-(a+d))/2)), int(st.lh)
}

// mixedLine is a line box holding text in several styles on one baseline (CSS inline
// layout): the box grows to fit the tallest part above and below the baseline.
func mixedLine(sts ...style) (above, height int) {
	below := 0
	for _, st := range sts {
		a, h := lineBox(st)
		above, below = max(above, a), max(below, h-a)
	}
	return above, above + below
}

// textOn draws s so that its baseline sits at baseline.
func (c *canvas) textOn(st style, x, baseline int, s string, v uint8) int {
	a, _ := lineBox(st)
	return c.text(st, x, baseline-a, s, v)
}

func measure(st style, s string) int {
	fc := faceFor(st)
	var x float64
	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			x += fixToF(fc.Kern(prev, r))
		}
		adv, _ := fc.GlyphAdvance(r)
		x += fixToF(adv) + st.spacing
		prev = r
	}
	return int(math.Round(x))
}

func fixToF(v fixed.Int26_6) float64 { return float64(v) / 64 }

// text draws s with its line box's top-left at (x, top). Each glyph snaps to a whole
// pixel so the pixel fonts stay crisp on e-ink.
func (c *canvas) text(st style, x, top int, s string, v uint8) int {
	fc := faceFor(st)
	above, _ := lineBox(st)
	src := image.NewUniform(color.Gray{v})
	var pen float64
	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			pen += fixToF(fc.Kern(prev, r))
		}
		dot := fixed.P(x+int(math.Round(pen)), top+above)
		dr, mask, mp, adv, ok := fc.Glyph(dot, r)
		if ok {
			draw.DrawMask(c.img, dr, src, image.Point{}, mask, mp, draw.Over)
		}
		pen += fixToF(adv) + st.spacing
		prev = r
	}
	return int(math.Round(pen))
}

// wrap breaks s into lines no wider than w, at spaces.
func wrap(st style, s string, w int) []string {
	var lines []string
	cur := ""
	for _, word := range strings.Fields(s) {
		try := word
		if cur != "" {
			try = cur + " " + word
		}
		if cur != "" && measure(st, try) > w {
			lines = append(lines, cur)
			cur = word
		} else {
			cur = try
		}
	}
	return append(lines, cur)
}

// icon draws a 16 x 16 pixel icon at a whole-number scale (1, 2, 4 or 8).
func (c *canvas) icon(name string, x, y, scale int, invert, filled bool) {
	rows, ok := icons[name]
	if !ok {
		return
	}
	for ry, row := range rows {
		for rx, ch := range row {
			switch {
			case ch == 'k' && invert:
				ch = 'w'
			case ch == 'w' && (invert || filled):
				ch = 'k'
			}
			var v uint8
			switch ch {
			case 'k':
				v = ink
			case 'w':
				v = paper
			case 'g':
				v = fog
			case 'm':
				v = mist
			default:
				continue
			}
			c.fill(x+rx*scale, y+ry*scale, x+(rx+1)*scale, y+(ry+1)*scale, v)
		}
	}
}

// ring draws a d x d circle with a 2px ink outline and a paper fill (title-bar dots).
func (c *canvas) ring(x, y, d int) {
	r := float64(d) / 2
	const ss = 4
	for py := 0; py < d; py++ {
		for px := 0; px < d; px++ {
			var sum float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/ss - r
					fy := float64(py) + (float64(sy)+0.5)/ss - r
					dist := math.Hypot(fx, fy)
					switch {
					case dist > r:
						sum += float64(c.img.GrayAt(x+px, y+py).Y)
					case dist > r-2:
						sum += float64(ink)
					default:
						sum += float64(paper)
					}
				}
			}
			c.img.SetGray(x+px, y+py, color.Gray{uint8(math.Round(sum / (ss * ss)))})
		}
	}
}

// frame draws a bordered box with a hard shadow-pop offset (6px, fog).
func (c *canvas) frame(x, y, w, h int) {
	c.fill(x+6, y+6, x+w+6, y+h+6, fog)
	c.fill(x, y, x+w, y+h, ink)
	c.fill(x+3, y+3, x+w-3, y+h-3, paper)
}

// window draws a Window (border-bold frame, mist title bar over a border rule, three dots)
// and returns its body rectangle.
func (c *canvas) window(x, y, w, h int, title string) image.Rectangle {
	c.frame(x, y, w, h)
	c.fill(x+3, y+3, x+w-3, y+35, mist)
	c.fill(x+3, y+35, x+w-3, y+37, ink)
	c.text(stLabel, x+15, y+11, title, ink)
	dx := x + w - 3 - 12 - 48
	for i := 0; i < 3; i++ {
		c.ring(dx+i*18, y+13, 12)
	}
	return image.Rect(x+3, y+37, x+w-3, y+h-3)
}

func (c *canvas) statusBar(x, y, w int, left, right string, battery int) {
	c.frame(x, y, w, 40)
	c.text(stLabel, x+15, y+12, left, ink)
	hx := x + w - 15 - 70
	full := 0
	if battery >= 0 {
		full = int(math.Ceil(float64(battery) / 25))
		if full > 4 {
			full = 4
		}
		for i := 0; i < 4; i++ {
			c.icon("heart", hx+i*18, y+12, 1, false, i < full)
		}
	} else {
		hx += 70 + 12
	}
	c.text(stLabel, hx-12-measure(stLabel, right), y+12, right, ink)
}

// key draws a pixel key (shadow-key: notched 2px ink outline and a fog ledge) and returns
// its rectangle. Width depends on the label.
func (c *canvas) key(x, y int, label, icon string, iconEnd, alignRight bool) image.Rectangle {
	tw := measure(stColLabel, label)
	w, h := 16+tw+8+16+16, 40
	if alignRight {
		x -= w
	}
	c.fill(x, y+6, x+w, y+h+6, fog)
	c.fill(x+2, y, x+w+2, y+h, ink)
	c.fill(x-2, y, x+w-2, y+h, ink)
	c.fill(x, y+2, x+w, y+h+2, ink)
	c.fill(x, y-2, x+w, y+h-2, ink)
	c.fill(x, y, x+w, y+h, paper)
	if iconEnd {
		c.text(stColLabel, x+16, y+8, label, ink)
		c.icon(icon, x+16+tw+8, y+12, 1, false, false)
	} else {
		c.icon(icon, x+16, y+12, 1, false, false)
		c.text(stColLabel, x+16+16+8, y+8, label, ink)
	}
	return image.Rect(x, y, x+w, y+h)
}

// finish snaps every pixel to the panel's 16-level gray ladder and applies the theme.
func (c *canvas) finish(night bool) *image.Gray {
	for i, v := range c.img.Pix {
		q := uint8(math.Round(float64(v)/17) * 17)
		if night {
			q = 255 - q
		}
		c.img.Pix[i] = q
	}
	return c.img
}

func hiStyle(s string) style {
	if len([]rune(s)) > 3 {
		return stNumLong
	}
	return stNumMd
}

// ---------- screens ----------

// Hit areas for the keys, padded so a thumb finds them.
var (
	hitDaysKey   image.Rectangle
	hitTodayKey  image.Rectangle
	hitStatusBar = image.Rect(0, 0, canvasW, 24+40+24) // hold here for 3s to exit
)

const footY = canvasH - 24 - 50 // the key row: 2px top margin, 40px key, 8px bottom margin

func renderMain(m *Model, night bool) *image.Gray {
	c := newCanvas()
	c.graphPaper()
	c.statusBar(24, 24, 552, m.DateLabel, m.Updated, m.Battery)

	// WEATHER.EXE: size the body from its content, like the CSS does.
	y := 24 + 40 + 16
	inner := 552 - 6 - 32
	readSt := stNumHero
	readout := m.Temp + "°"
	if len([]rune(readout)) > 3 {
		readSt = stNumHeroXL
	}
	wordLines := wrap(stDisplay, m.Word, inner-128)
	feelsA, feelsH := mixedLine(stLabel, stNumSm)
	leftH := 112 + 12 + 60*len(wordLines) + 12 + feelsH
	nowH := max(leftH, 128)
	_, valueH := mixedLine(stNumMd, stLabel)
	statH := 16 + 8 + valueH
	bodyH := nowH + 16 + 2 + 16 + statH
	winH := 3 + 34 + 16 + bodyH + 16 + 3
	b := c.window(24, y, 552, winH, "WEATHER.EXE")
	bx, by := b.Min.X+16, b.Min.Y+16

	c.text(readSt, bx, by, readout, ink)
	wy := by + 112 + 12
	for i, ln := range wordLines {
		c.text(stDisplay, bx, wy+60*i, ln, ink)
	}
	fy := wy + 60*len(wordLines) + 12
	fw := c.textOn(stLabel, bx, fy+feelsA, "FEELS LIKE ", slate)
	c.textOn(stNumSm, bx+fw, fy+feelsA, m.Feels+"°", ink)
	c.icon(m.Icon, bx+inner-128, by, 8, false, false)

	ry := by + nowH + 16
	c.fill(bx, ry, bx+inner, ry+2, ink)
	sy := ry + 2 + 16
	colW := (inner - 16) / 2
	c.stat(bx, sy, statH, "thermometer", "HIGH TODAY", m.High+"°", "")
	c.stat(bx+colW+16, sy, statH, m.SunIcon, m.SunLabel, m.SunTime, m.SunUnit)

	// HOURS.EXE grows into the spare height above the key row.
	hy := y + winH + 16
	hb := c.window(24, hy, 552, footY-16-hy, "HOURS.EXE")
	c.hourColumns(hb, m.Hours)

	r := c.key(576-2, footY+2, "5 DAYS", "next", true, true)
	hitDaysKey = r.Inset(-24)
	return c.finish(night)
}

// stat draws a Stat centred in a row rowH tall (a unit makes the value line taller).
func (c *canvas) stat(x, y, rowH int, icon, label, value, unit string) {
	c.icon(icon, x, y+(rowH-32)/2, 2, false, false)
	tx := x + 32 + 12
	// The digits and the caps unit share a baseline.
	va, vh := lineBox(stNumMd)
	if unit != "" {
		va, vh = mixedLine(stNumMd, stLabel)
	}
	y += (rowH - (16 + 8 + vh)) / 2
	c.text(stLabel, tx, y, label, slate)
	vw := c.textOn(stNumMd, tx, y+16+8+va, value, ink)
	if unit != "" {
		c.textOn(stLabel, tx+vw+8, y+16+8+va, unit, slate)
	}
}

func colEdges(b image.Rectangle, n, i int) (int, int) {
	w := float64(b.Dx())
	return b.Min.X + int(math.Round(float64(i)*w/float64(n))), b.Min.X + int(math.Round(float64(i+1)*w/float64(n)))
}

func (c *canvas) hourColumns(b image.Rectangle, hours []Hour) {
	n := len(hours)
	for i, hr := range hours {
		l, r := colEdges(b, n, i)
		if i > 0 {
			c.fill(l, b.Min.Y, l+2, b.Max.Y, ink)
			l += 2
		}
		center := float64(l+r) / 2
		at := func(w int) int { return int(math.Round(center - float64(w)/2)) }

		hi := hr.Temp + "°"
		contentH := 24 + 8 + 64 + 8 + 32 + 8 + 16
		y := b.Min.Y + 12 + (b.Dy()-24-contentH)/2
		c.text(stColLabel, at(measure(stColLabel, hr.Label)), y, hr.Label, ink)
		y += 24 + 8
		c.icon(hr.Icon, at(64), y, 4, false, false)
		y += 64 + 8
		hs := hiStyle(hi)
		c.text(hs, at(measure(hs, hi)), y, hi, ink)
		y += 32 + 8
		if hr.Rain >= 10 {
			txt := itoa(hr.Rain) + "%"
			w := 16 + 4 + measure(stNumSm, txt)
			x := at(w)
			c.icon("drop", x, y, 1, false, false)
			c.text(stNumSm, x+20, y, txt, slate)
		}
	}
}

func renderDays(m *Model, night bool) *image.Gray {
	c := newCanvas()
	c.graphPaper()
	c.statusBar(24, 24, 552, m.DateLabel, m.Updated, m.Battery)
	y := 24 + 40 + 16
	b := c.window(24, y, 552, footY-16-y, "5DAYS.EXE")
	c.dayRows(b, m.Days)
	r := c.key(24+2, footY+2, "TODAY", "back", false, false)
	hitTodayKey = r.Inset(-24)
	return c.finish(night)
}

func (c *canvas) dayRows(b image.Rectangle, days []Day) {
	n := len(days)
	for i, d := range days {
		t := b.Min.Y + int(math.Round(float64(i*b.Dy())/float64(n)))
		bt := b.Min.Y + int(math.Round(float64((i+1)*b.Dy())/float64(n)))
		today := i == 0
		fg, sec := ink, slate
		if today {
			c.fill(b.Min.X, t, b.Max.X, bt, ink)
			fg, sec = paper, paper
		}
		if i > 0 {
			c.fill(b.Min.X, t, b.Max.X, t+2, ink)
			t += 2
		}
		ct, ch := t+12, bt-12-(t+12)
		mid := func(h int) int { return ct + (ch-h)/2 }

		cx, cw := b.Min.X+16, b.Dx()-32
		wordW := cw - (64 + 64 + 72 + 96 + 72) - 5*8
		xIcon := cx + 64 + 8
		xWord := xIcon + 64 + 8
		xRain := xWord + wordW + 8
		hiRight := xRain + 72 + 8 + 96
		loRight := cx + cw

		c.text(stColLabel, cx, mid(24), d.Label, fg)
		c.icon(d.Icon, xIcon, mid(64), 4, today, false)
		lines := wrap(stWord, d.Word, wordW)
		for j, ln := range lines {
			c.text(stWord, xWord, mid(28*len(lines))+28*j, ln, fg)
		}
		if d.Rain >= 10 {
			ry := mid(16)
			c.icon("drop", xRain, ry, 1, today, false)
			c.text(stNumSm, xRain+20, ry, itoa(d.Rain)+"%", sec)
		}
		hi := d.Hi + "°"
		hs := hiStyle(hi)
		c.text(hs, hiRight-measure(hs, hi), mid(32), hi, fg)
		lo := d.Lo + "°"
		c.text(stNumSmPlus, loRight-measure(stNumSmPlus, lo), mid(24), lo, sec)
	}
}

// renderMessage is a one-window screen: the alert when there is no weather to show yet,
// or a plain window for the goodbye.
func renderMessage(m *Model, title, icon, msg string, alert, night bool) *image.Gray {
	c := newCanvas()
	c.graphPaper()
	c.statusBar(24, 24, 552, m.DateLabel, m.Updated, m.Battery)
	y := 24 + 40 + 16
	lines := wrap(stBody, msg, 552-6-32-64-16)
	h := 3 + 34 + 16 + max(64, 32*len(lines)) + 16 + 3
	if alert {
		c.frame(24, y, 552, h)
		c.fill(27, y+3, 573, y+35, ink)
		c.text(stLabel, 39, y+11, title, paper)
		c.fill(27, y+35, 573, y+37, ink)
	} else {
		c.window(24, y, 552, h, title)
	}
	c.icon(icon, 43, y+37+16, 4, false, false)
	for i, ln := range lines {
		c.text(stBody, 43+64+16, y+37+16+32*i, ln, ink)
	}
	return c.finish(night)
}
