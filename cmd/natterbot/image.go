package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync"

	"github.com/nguyenquocanhz/telegramMTP/mtp"
)

// demoPhoto returns a small generated gradient so /photo works without
// shipping an image file. It is rendered once and reused.
var demoPhoto = sync.OnceValue(func() mtp.Photo {
	const w, h = 640, 360
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(40 + 180*x/w),
				G: uint8(90 + 120*y/h),
				B: uint8(220 - 120*x/w),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // encoding an in-memory RGBA image cannot fail
	}
	return mtp.PhotoBytes("demo.png", buf.Bytes())
})
