package webserver

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

var icon192, icon512 []byte

func init() {
	icon192 = makeIcon(192)
	icon512 = makeIcon(512)
}

func makeIcon(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	bg := color.NRGBA{0x11, 0x11, 0x11, 0xff}
	fg := color.NRGBA{0xeb, 0x8d, 0x28, 0xff}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, bg)
		}
	}

	cx, cy := size/2, size/2
	r := float64(size) / 3
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if (dx*dx+dy*dy)/(r*r) <= 1 {
				img.Set(x, y, fg)
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
