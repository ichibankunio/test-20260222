package game

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	bulletsPerBatch = 4000
)

const bulletShaderSource = `//kage:unit pixels

package main

var InnerColor vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	dist2 := dot(srcPos, srcPos)
	if dist2 > 1.0 {
		return vec4(0)
	}

	innerRadius2 := custom.x
	if dist2 <= innerRadius2 {
		return InnerColor
	}
	return color
}
`

type bulletShaderRenderer struct {
	once     sync.Once
	shader   *ebiten.Shader
	initErr  error
	vertices []ebiten.Vertex
	indices  []uint16
	op       ebiten.DrawTrianglesShaderOptions
}

var globalBulletShaderRenderer bulletShaderRenderer

func (r *bulletShaderRenderer) init() {
	r.shader, r.initErr = ebiten.NewShader([]byte(bulletShaderSource))
	if r.initErr != nil {
		r.initErr = fmt.Errorf("new bullet shader: %w", r.initErr)
		return
	}
	ic := colorToFloat32(enemyBulletIn)
	r.op.Uniforms = map[string]any{
		"InnerColor": []float32{ic[0], ic[1], ic[2], ic[3]},
	}
	r.op.AntiAlias = false
}

func (r *bulletShaderRenderer) draw(screen *ebiten.Image, bullets []projectile) bool {
	r.once.Do(r.init)
	if r.initErr != nil {
		return false
	}

	for start := 0; start < len(bullets); start += bulletsPerBatch {
		end := start + bulletsPerBatch
		if end > len(bullets) {
			end = len(bullets)
		}
		r.drawBatch(screen, bullets[start:end])
	}
	return true
}

func (r *bulletShaderRenderer) drawBatch(screen *ebiten.Image, bullets []projectile) {
	if len(bullets) == 0 {
		return
	}

	vertexCount := len(bullets) * 4
	indexCount := len(bullets) * 6
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

	outer := colorToFloat32(enemyBullet)
	for i, b := range bullets {
		vi := i * 4
		ii := i * 6
		cx := float32(b.x)
		cy := float32(b.y)
		half := float32(b.radius)
		innerRadius2 := float32(0.38 * 0.38)

		r.vertices[vi+0] = bulletVertex(cx-half, cy-half, -1, -1, outer, innerRadius2)
		r.vertices[vi+1] = bulletVertex(cx+half, cy-half, +1, -1, outer, innerRadius2)
		r.vertices[vi+2] = bulletVertex(cx-half, cy+half, -1, +1, outer, innerRadius2)
		r.vertices[vi+3] = bulletVertex(cx+half, cy+half, +1, +1, outer, innerRadius2)

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

func bulletVertex(dstX, dstY, srcX, srcY float32, outer [4]float32, innerRadius2 float32) ebiten.Vertex {
	return ebiten.Vertex{
		DstX:    dstX,
		DstY:    dstY,
		SrcX:    srcX,
		SrcY:    srcY,
		ColorR:  outer[0],
		ColorG:  outer[1],
		ColorB:  outer[2],
		ColorA:  outer[3],
		Custom0: innerRadius2,
	}
}

func colorToFloat32(c interface{ RGBA() (r, g, b, a uint32) }) [4]float32 {
	r, g, b, a := c.RGBA()
	return [4]float32{
		float32(r) / 0xffff,
		float32(g) / 0xffff,
		float32(b) / 0xffff,
		float32(a) / 0xffff,
	}
}
