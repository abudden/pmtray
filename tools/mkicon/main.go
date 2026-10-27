// Command mkicon draws the application icon (a key on a rounded square)
// and writes it as a PNG. Run with: go run ./tools/mkicon internal/ui/assets/icon.png
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

const (
	size = 256
	ss   = 4 // supersampling factor per axis
)

// inside reports whether the point (x, y), in 0..1 coordinates, is part of
// the background (1) or the key (2), or neither (0).
func inside(x, y float64) int {
	// Rounded square background.
	const r = 0.2
	dx := math.Max(math.Abs(x-0.5)-(0.5-r), 0)
	dy := math.Max(math.Abs(y-0.5)-(0.5-r), 0)
	if dx*dx+dy*dy > r*r {
		return 0
	}
	// Key, drawn diagonally: rotate the point by 45 degrees around the centre.
	cx, cy := x-0.5, y-0.5
	u := (cx - cy) / math.Sqrt2 // along the key
	v := (cx + cy) / math.Sqrt2 // across the key
	// Bow: a ring centred at u = -0.2.
	d := math.Hypot(u+0.2, v)
	if d < 0.17 && d > 0.075 {
		return 2
	}
	// Shaft.
	if u > -0.04 && u < 0.36 && math.Abs(v) < 0.045 {
		return 2
	}
	// Teeth.
	if v > 0 && v < 0.13 && ((u > 0.2 && u < 0.26) || (u > 0.3 && u < 0.36)) {
		return 2
	}
	return 1
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: mkicon output.png")
	}
	bg := color.NRGBA{0x1f, 0x6f, 0xb5, 0xff}
	fg := color.NRGBA{0xff, 0xd5, 0x4a, 0xff}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/ss) / size
					y := (float64(py) + (float64(sy)+0.5)/ss) / size
					var c color.NRGBA
					switch inside(x, y) {
					case 0:
						continue
					case 1:
						c = bg
					case 2:
						c = fg
					}
					r += float64(c.R)
					g += float64(c.G)
					b += float64(c.B)
					a++
				}
			}
			if a == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				uint8(r / a), uint8(g / a), uint8(b / a), uint8(a / (ss * ss) * 255),
			})
		}
	}
	f, err := os.Create(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}
