package game

import (
	"fmt"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	portraitPixelsPerBatch = 8000
)

const portraitPixelShaderSource = `//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	if srcPos.x < 0.0 || srcPos.x > 1.0 || srcPos.y < 0.0 || srcPos.y > 1.0 {
		return vec4(0)
	}
	return color
}
`

type portraitDrawPixel struct {
	x    float64
	y    float64
	size float64
	col  color.RGBA
}

type portraitPixelShaderRenderer struct {
	once     sync.Once
	shader   *ebiten.Shader
	initErr  error
	vertices []ebiten.Vertex
	indices  []uint16
	op       ebiten.DrawTrianglesShaderOptions
}

var globalPortraitPixelShaderRenderer portraitPixelShaderRenderer

func (r *portraitPixelShaderRenderer) init() {
	r.shader, r.initErr = ebiten.NewShader([]byte(portraitPixelShaderSource))
	if r.initErr != nil {
		r.initErr = fmt.Errorf("new portrait pixel shader: %w", r.initErr)
		return
	}
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

		r.vertices[vi+0] = portraitPixelVertex(x, y, 0, 0, col)
		r.vertices[vi+1] = portraitPixelVertex(x+size, y, 1, 0, col)
		r.vertices[vi+2] = portraitPixelVertex(x, y+size, 0, 1, col)
		r.vertices[vi+3] = portraitPixelVertex(x+size, y+size, 1, 1, col)

		base := uint16(vi)
		r.indices[ii+0] = base + 0
		r.indices[ii+1] = base + 1
		r.indices[ii+2] = base + 2
		r.indices[ii+3] = base + 1
		r.indices[ii+4] = base + 2
		r.indices[ii+5] = base + 3
	}

	screen.DrawTrianglesShader(r.vertices, r.indices, r.shader, &r.op)
}

func portraitPixelVertex(dstX, dstY, srcX, srcY float32, col [4]float32) ebiten.Vertex {
	return ebiten.Vertex{
		DstX:   dstX,
		DstY:   dstY,
		SrcX:   srcX,
		SrcY:   srcY,
		ColorR: col[0],
		ColorG: col[1],
		ColorB: col[2],
		ColorA: col[3],
	}
}
