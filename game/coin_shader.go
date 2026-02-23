package game

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	coinsPerBatch = 6000
)

const coinShaderSource = `//kage:unit pixels

package main

var CoinColor vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	if dot(srcPos, srcPos) > 1.0 {
		return vec4(0)
	}
	return CoinColor
}
`

type coinShaderRenderer struct {
	once     sync.Once
	shader   *ebiten.Shader
	initErr  error
	vertices []ebiten.Vertex
	indices  []uint16
	op       ebiten.DrawTrianglesShaderOptions
}

var globalCoinShaderRenderer coinShaderRenderer

func (r *coinShaderRenderer) init() {
	r.shader, r.initErr = ebiten.NewShader([]byte(coinShaderSource))
	if r.initErr != nil {
		r.initErr = fmt.Errorf("new coin shader: %w", r.initErr)
		return
	}
	col := colorToFloat32(coinMain)
	r.op.Uniforms = map[string]any{
		"CoinColor": []float32{col[0], col[1], col[2], col[3]},
	}
	r.op.AntiAlias = false
}

func (r *coinShaderRenderer) draw(screen *ebiten.Image, coins []projectile) bool {
	r.once.Do(r.init)
	if r.initErr != nil || len(coins) == 0 {
		return false
	}
	for start := 0; start < len(coins); start += coinsPerBatch {
		end := start + coinsPerBatch
		if end > len(coins) {
			end = len(coins)
		}
		r.drawBatch(screen, coins[start:end])
	}
	return true
}

func (r *coinShaderRenderer) drawBatch(screen *ebiten.Image, coins []projectile) {
	vertexCount := len(coins) * 4
	indexCount := len(coins) * 6
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

	for i, c := range coins {
		vi := i * 4
		ii := i * 6
		cx := float32(c.x)
		cy := float32(c.y)
		half := float32(c.radius)

		r.vertices[vi+0] = coinVertex(cx-half, cy-half, -1, -1)
		r.vertices[vi+1] = coinVertex(cx+half, cy-half, +1, -1)
		r.vertices[vi+2] = coinVertex(cx-half, cy+half, -1, +1)
		r.vertices[vi+3] = coinVertex(cx+half, cy+half, +1, +1)

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

func coinVertex(dstX, dstY, srcX, srcY float32) ebiten.Vertex {
	return ebiten.Vertex{
		DstX:   dstX,
		DstY:   dstY,
		SrcX:   srcX,
		SrcY:   srcY,
		ColorR: 1,
		ColorG: 1,
		ColorB: 1,
		ColorA: 1,
	}
}
