package game

import (
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	portraitPixelsPerBatch = 8000
)

type portraitDrawPixel struct {
	x    float64
	y    float64
	size float64
	col  color.RGBA
}

type portraitPixelShaderRenderer struct {
	once     sync.Once
	src      *ebiten.Image
	initErr  error
	vertices []ebiten.Vertex
	indices  []uint16
	op       ebiten.DrawTrianglesOptions
}

var globalPortraitPixelShaderRenderer portraitPixelShaderRenderer

func (r *portraitPixelShaderRenderer) init() {
	r.src = ebiten.NewImage(1, 1)
	r.src.Fill(color.White)
	r.op.Filter = ebiten.FilterNearest
	r.op.AntiAlias = false
}

func (r *portraitPixelShaderRenderer) draw(screen *ebiten.Image, pixels []portraitDrawPixel) bool {
	r.once.Do(r.init)
	if r.initErr != nil || len(pixels) == 0 {
		return false
	}
	for start := 0; start < len(pixels); start += portraitPixelsPerBatch {
		end := start + portraitPixelsPerBatch
		if end > len(pixels) {
			end = len(pixels)
		}
		r.drawBatch(screen, pixels[start:end])
	}
	return true
}

func (r *portraitPixelShaderRenderer) drawBatch(screen *ebiten.Image, pixels []portraitDrawPixel) {
	vertexCount := len(pixels) * 4
	indexCount := len(pixels) * 6
	if cap(r.vertices) < vertexCount {
		r.vertices = make([]ebiten.Vertex, vertexCount)
	} else {
		r.vertices = r.vertices[:vertexCount]
	}
	if cap(r.indices) < indexCount {
		r.indices = make([]uint16, indexCount)
	} else {
		r.indices = r.indices[:indexCount]
	}

	for i, px := range pixels {
		vi := i * 4
		ii := i * 6
		x := float32(px.x)
		y := float32(px.y)
		size := float32(px.size)
		col := colorToFloat32(px.col)

		r.vertices[vi+0] = portraitPixelVertex(x, y, col)
		r.vertices[vi+1] = portraitPixelVertex(x+size, y, col)
		r.vertices[vi+2] = portraitPixelVertex(x, y+size, col)
		r.vertices[vi+3] = portraitPixelVertex(x+size, y+size, col)

		base := uint16(vi)
		r.indices[ii+0] = base + 0
		r.indices[ii+1] = base + 1
		r.indices[ii+2] = base + 2
		r.indices[ii+3] = base + 1
		r.indices[ii+4] = base + 2
		r.indices[ii+5] = base + 3
	}

	screen.DrawTriangles(r.vertices, r.indices, r.src, &r.op)
}

func portraitPixelVertex(dstX, dstY float32, col [4]float32) ebiten.Vertex {
	return ebiten.Vertex{
		DstX:   dstX,
		DstY:   dstY,
		// Sample the center texel of the 1x1 white source image.
		SrcX:   0.5,
		SrcY:   0.5,
		ColorR: col[0],
		ColorG: col[1],
		ColorB: col[2],
		ColorA: col[3],
	}
}
