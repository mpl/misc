/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
)

// Resize returns a scaled copy of the image slice r of m.
// The returned image has width w and height h.
func resize(m image.Image, r image.Rectangle, w, h int) image.Image {
	if w < 0 || h < 0 {
		return nil
	}
	if w == 0 || h == 0 || r.Dx() <= 0 || r.Dy() <= 0 {
		return image.NewRGBA64(image.Rect(0, 0, w, h))
	}
	switch m := m.(type) {
	case *image.RGBA:
		return resizeRGBA(m, r, w, h)
	case *image.YCbCr:
		if m, ok := resizeYCbCr(m, r, w, h); ok {
			return m
		}
	}
	ww, hh := uint64(w), uint64(h)
	dx, dy := uint64(r.Dx()), uint64(r.Dy())
	// The scaling algorithm is to nearest-neighbor magnify the dx * dy source
	// to a (ww*dx) * (hh*dy) intermediate image and then minify the intermediate
	// image back down to a ww * hh destination with a simple box filter.
	// The intermediate image is implied, we do not physically allocate a slice
	// of length ww*dx*hh*dy.
	// For example, consider a 4*3 source image. Label its pixels from a-l:
	//	abcd
	//	efgh
	//	ijkl
	// To resize this to a 3*2 destination image, the intermediate is 12*6.
	// Whitespace has been added to delineate the destination pixels:
	//	aaab bbcc cddd
	//	aaab bbcc cddd
	//	eeef ffgg ghhh
	//
	//	eeef ffgg ghhh
	//	iiij jjkk klll
	//	iiij jjkk klll
	// Thus, the 'b' source pixel contributes one third of its value to the
	// (0, 0) destination pixel and two thirds to (1, 0).
	// The implementation is a two-step process. First, the source pixels are
	// iterated over and each source pixel's contribution to 1 or more
	// destination pixels are summed. Second, the sums are divided by a scaling
	// factor to yield the destination pixels.
	// TODO: By interleaving the two steps, instead of doing all of
	// step 1 first and all of step 2 second, we could allocate a smaller sum
	// slice of length 4*w*2 instead of 4*w*h, although the resultant code
	// would become more complicated.
	n, sum := dx*dy, make([]uint64, 4*w*h)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			// Get the source pixel.
			r32, g32, b32, a32 := m.At(x, y).RGBA()
			r64 := uint64(r32)
			g64 := uint64(g32)
			b64 := uint64(b32)
			a64 := uint64(a32)
			// Spread the source pixel over 1 or more destination rows.
			py := uint64(y-r.Min.Y) * hh
			for remy := hh; remy > 0; {
				qy := dy - (py % dy)
				if qy > remy {
					qy = remy
				}
				// Spread the source pixel over 1 or more destination columns.
				px := uint64(x-r.Min.X) * ww
				index := 4 * ((py/dy)*ww + (px / dx))
				for remx := ww; remx > 0; {
					qx := dx - (px % dx)
					if qx > remx {
						qx = remx
					}
					sum[index+0] += r64 * qx * qy
					sum[index+1] += g64 * qx * qy
					sum[index+2] += b64 * qx * qy
					sum[index+3] += a64 * qx * qy
					index += 4
					px += qx
					remx -= qx
				}
				py += qy
				remy -= qy
			}
		}
	}
	return average(sum, w, h, n*0x0101)
}

// average convert the sums to averages and returns the result.
func average(sum []uint64, w, h int, n uint64) image.Image {
	ret := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*ret.Stride + x*4
			j := 4 * (y*w + x)
			ret.Pix[i+0] = uint8(sum[j+0] / n)
			ret.Pix[i+1] = uint8(sum[j+1] / n)
			ret.Pix[i+2] = uint8(sum[j+2] / n)
			ret.Pix[i+3] = uint8(sum[j+3] / n)
		}
	}
	return ret
}

// resizeYCbCr returns a scaled copy of the YCbCr image slice r of m.
// The returned image has width w and height h.
func resizeYCbCr(m *image.YCbCr, r image.Rectangle, w, h int) (image.Image, bool) {
	var verticalRes int
	switch m.SubsampleRatio {
	case image.YCbCrSubsampleRatio420:
		verticalRes = 2
	case image.YCbCrSubsampleRatio422:
		verticalRes = 1
	default:
		return nil, false
	}
	ww, hh := uint64(w), uint64(h)
	dx, dy := uint64(r.Dx()), uint64(r.Dy())
	// See comment in Resize.
	n, sum := dx*dy, make([]uint64, 4*w*h)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		Y := m.Y[y*m.YStride:]
		Cb := m.Cb[y/verticalRes*m.CStride:]
		Cr := m.Cr[y/verticalRes*m.CStride:]
		for x := r.Min.X; x < r.Max.X; x++ {
			// Get the source pixel.
			r8, g8, b8 := color.YCbCrToRGB(Y[x], Cb[x/2], Cr[x/2])
			r64 := uint64(r8)
			g64 := uint64(g8)
			b64 := uint64(b8)
			// Spread the source pixel over 1 or more destination rows.
			py := uint64(y-r.Min.Y) * hh
			for remy := hh; remy > 0; {
				qy := dy - (py % dy)
				if qy > remy {
					qy = remy
				}
				// Spread the source pixel over 1 or more destination columns.
				px := uint64(x-r.Min.X) * ww
				index := 4 * ((py/dy)*ww + (px / dx))
				for remx := ww; remx > 0; {
					qx := dx - (px % dx)
					if qx > remx {
						qx = remx
					}
					qxy := qx * qy
					sum[index+0] += r64 * qxy
					sum[index+1] += g64 * qxy
					sum[index+2] += b64 * qxy
					sum[index+3] += 0xFFFF * qxy
					index += 4
					px += qx
					remx -= qx
				}
				py += qy
				remy -= qy
			}
		}
	}
	return average(sum, w, h, n), true
}

// resizeRGBA returns a scaled copy of the RGBA image slice r of m.
// The returned image has width w and height h.
func resizeRGBA(m *image.RGBA, r image.Rectangle, w, h int) image.Image {
	ww, hh := uint64(w), uint64(h)
	dx, dy := uint64(r.Dx()), uint64(r.Dy())
	// See comment in Resize.
	n, sum := dx*dy, make([]uint64, 4*w*h)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		pix := m.Pix[(y-m.Rect.Min.Y)*m.Stride:]
		for x := r.Min.X; x < r.Max.X; x++ {
			// Get the source pixel.
			p := pix[(x-m.Rect.Min.X)*4:]
			r64 := uint64(p[0])
			g64 := uint64(p[1])
			b64 := uint64(p[2])
			a64 := uint64(p[3])
			// Spread the source pixel over 1 or more destination rows.
			py := uint64(y-r.Min.Y) * hh
			for remy := hh; remy > 0; {
				qy := dy - (py % dy)
				if qy > remy {
					qy = remy
				}
				// Spread the source pixel over 1 or more destination columns.
				px := uint64(x-r.Min.X) * ww
				index := 4 * ((py/dy)*ww + (px / dx))
				for remx := ww; remx > 0; {
					qx := dx - (px % dx)
					if qx > remx {
						qx = remx
					}
					qxy := qx * qy
					sum[index+0] += r64 * qxy
					sum[index+1] += g64 * qxy
					sum[index+2] += b64 * qxy
					sum[index+3] += a64 * qxy
					index += 4
					px += qx
					remx -= qx
				}
				py += qy
				remy -= qy
			}
		}
	}
	return average(sum, w, h, n)
}

// Resample returns a resampled copy of the image slice r of m.
// The returned image has width w and height h.
func resample(m image.Image, r image.Rectangle, w, h int) image.Image {
	if w < 0 || h < 0 {
		return nil
	}
	if w == 0 || h == 0 || r.Dx() <= 0 || r.Dy() <= 0 {
		return image.NewRGBA64(image.Rect(0, 0, w, h))
	}
	curw, curh := r.Dx(), r.Dy()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Get a source pixel.
			subx := x * curw / w
			suby := y * curh / h
			r32, g32, b32, a32 := m.At(subx, suby).RGBA()
			r := uint8(r32 >> 8)
			g := uint8(g32 >> 8)
			b := uint8(b32 >> 8)
			a := uint8(a32 >> 8)
			img.SetRGBA(x, y, color.RGBA{r, g, b, a})
		}
	}
	return img
}

type subImager interface {
	SubImage(image.Rectangle) image.Image
}

func squareImage(i image.Image) image.Image {
	si, ok := i.(subImager)
	if !ok {
		log.Fatalf("image %T isn't a subImager", i)
	}
	b := i.Bounds()
	if b.Dx() > b.Dy() {
		thin := (b.Dx() - b.Dy()) / 2
		newB := b
		newB.Min.X += thin
		newB.Max.X -= thin
		return si.SubImage(newB)
	}
	thin := (b.Dy() - b.Dx()) / 2
	newB := b
	newB.Min.Y += thin
	newB.Max.Y -= thin
	return si.SubImage(newB)
}

func scaleImage(buf *bytes.Buffer, mw, mh int) (format string, err error) {
	i, format, err := image.Decode(bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		return format, err
	}
	b := i.Bounds()

	useBytesUnchanged := true
	wantSquare := false

	isSquare := b.Dx() == b.Dy()
	if wantSquare && !isSquare {
		useBytesUnchanged = false
		i = squareImage(i)
		b = i.Bounds()
	}

	// only do downscaling, otherwise just serve the original image
	if mw < b.Dx() || mh < b.Dy() {
		useBytesUnchanged = false

		const huge = 2400
		// If it's gigantic, it's more efficient to downsample first
		// and then resize; resizing will smooth out the roughness.
		// (trusting the moustachio guys on that one).
		if b.Dx() > huge || b.Dy() > huge {
			w, h := mw*2, mh*2
			if b.Dx() > b.Dy() {
				w = b.Dx() * h / b.Dy()
			} else {
				h = b.Dy() * w / b.Dx()
			}
			i = resample(i, i.Bounds(), w, h)
			b = i.Bounds()
		}
		// conserve proportions. use the smallest of the two as the decisive one.
		if mw > mh {
			mw = b.Dx() * mh / b.Dy()
		} else {
			mh = b.Dy() * mw / b.Dx()
		}
	}

	if !useBytesUnchanged {
		i = resize(i, b, mw, mh)
		// Encode as a new image
		buf.Reset()
		switch format {
		case "jpeg":
			err = jpeg.Encode(buf, i, nil)
		default:
			err = png.Encode(buf, i)
		}
		if err != nil {
			return format, err
		}
	}
	return format, nil
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("give an image file as arg plz, kthxbai.")
	}

	img := flag.Arg(0)
	f, err := os.Open(img)
	if err != nil {
		log.Fatal(err)
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	_, err = io.Copy(buf, f)
	if err != nil {
		log.Fatal("error reading image %s: %v", img, err)
	}

	_, err = scaleImage(buf, 300, 200)

	g, err := os.Create(img + "-resized")
	if err != nil {
		log.Fatal(err)
	}
	_, err = io.Copy(g, buf)
	if err != nil {
		log.Fatal("error writing image %s: %v", img + "-resized", err)
	}

}
