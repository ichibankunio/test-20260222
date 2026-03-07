package game

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ichibankunio/test-20260222/game/mathutil"
)

const (
	particlesPerBatch = 3000
)

const impactParticleShaderSource = `//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	dist := length(srcPos)
	if dist >= 1.0 {
		return vec4(0)
	}

	life := clamp(custom.x, 0.0, 1.0)
	turbulence := custom.y
	coreRadius := clamp(custom.z, 0.08, 0.8)
	hot := 1.0 - life

	core := clamp((coreRadius-dist)/(coreRadius+0.0001), 0.0, 1.0)
	halo := clamp((1.0-dist)*1.3, 0.0, 1.0)
	streak := sin(atan2(srcPos.y, srcPos.x)*7.0 + turbulence*18.0 + hot*8.0)
	flare := pow(clamp(streak*0.5+0.5, 0.0, 1.0), 3.0) * pow(clamp(1.0-dist, 0.0, 1.0), 1.7)

	alpha := clamp((halo*0.72+core*0.28+flare*0.65)*life, 0.0, 1.0)
	return vec4(color.rgb*alpha, alpha)
}
`

type impactParticle struct {
	x         float64
	y         float64
	vx        float64
	vy        float64
	radius    float64
	life      int
	maxLife   int
	turb      float64
	coreRatio float64
	col       [4]float32
}

type impactParticleSystem struct {
	items []impactParticle
}

func (p *impactParticleSystem) reset() {
	p.items = p.items[:0]
}

func (p *impactParticleSystem) update() {
	n := 0
	for i := range p.items {
		it := p.items[i]
		it.life--
		if it.life <= 0 {
			continue
		}
		it.vx *= 0.98
		it.vy = it.vy*0.98 + 0.015
		it.x += it.vx
		it.y += it.vy
		if it.x < -20 || it.x > ScreenWidth+20 || it.y < -20 || it.y > ScreenHeight+20 {
			continue
		}
		p.items[n] = it
		n++
	}
	p.items = p.items[:n]
}

func (p *impactParticleSystem) draw(screen *ebiten.Image) {
	globalImpactParticleRenderer.draw(screen, p.items)
}

func (p *impactParticleSystem) spawnPlayerBurst(x, y float64) {
	playerCol := colorToFloat32(playerMain)
	p.spawnRadial(x, y, 56, 0.35, 3.0, 1.1, 2.8, 24, 42, [2][4]float32{playerCol, playerCol})
	p.spawnRadial(x, y, 22, 2.8, 5.0, 0.9, 1.7, 16, 28, [2][4]float32{playerCol, playerCol})
}

func (p *impactParticleSystem) spawnCoinPickup(x, y float64) {
	coinCol := colorToFloat32(coinMain)
	p.spawnRadial(x, y, 18, 0.25, 2.3, 0.9, 2.1, 18, 34, [2][4]float32{coinCol, coinCol})
	p.spawnRadial(x, y, 10, 1.8, 3.6, 0.8, 1.5, 12, 24, [2][4]float32{coinCol, coinCol})
}

func (p *impactParticleSystem) spawnBulletImpact(x, y float64) {
	bulletCol := colorToFloat32(enemyBullet)
	p.spawnRadial(x, y, 10, 0.2, 1.4, 0.7, 1.5, 10, 18, [2][4]float32{bulletCol, bulletCol})
}

func (p *impactParticleSystem) spawnRadial(x, y float64, count int, speedMin, speedMax, radiusMin, radiusMax float64, lifeMin, lifeMax int, palette [2][4]float32) {
	if count <= 0 {
		return
	}
	for i := 0; i < count; i++ {
		ang := rand.Float64() * 2 * math.Pi
		spd := speedMin + rand.Float64()*(speedMax-speedMin)
		radius := radiusMin + rand.Float64()*(radiusMax-radiusMin)
		maxLife := lifeMin + rand.IntN(max(1, lifeMax-lifeMin+1))
		col := palette[rand.IntN(len(palette))]
		p.items = append(p.items, impactParticle{
			x:         x,
			y:         y,
			vx:        math.Cos(ang) * spd,
			vy:        math.Sin(ang) * spd,
			radius:    radius,
			life:      maxLife,
			maxLife:   maxLife,
			turb:      rand.Float64(),
			coreRatio: 0.28 + rand.Float64()*0.3,
			col:       col,
		})
	}
}

type impactParticleRenderer struct {
	once     sync.Once
	shader   *ebiten.Shader
	initErr  error
	vertices []ebiten.Vertex
	indices  []uint16
	op       ebiten.DrawTrianglesShaderOptions
}

var globalImpactParticleRenderer impactParticleRenderer

func (r *impactParticleRenderer) init() {
	r.shader, r.initErr = ebiten.NewShader([]byte(impactParticleShaderSource))
	if r.initErr != nil {
		r.initErr = fmt.Errorf("new impact particle shader: %w", r.initErr)
		return
	}
}

func (r *impactParticleRenderer) draw(screen *ebiten.Image, particles []impactParticle) {
	r.once.Do(r.init)
	if r.initErr != nil || len(particles) == 0 {
		return
	}
	for start := 0; start < len(particles); start += particlesPerBatch {
		end := start + particlesPerBatch
		if end > len(particles) {
			end = len(particles)
		}
		r.drawBatch(screen, particles[start:end])
	}
}

func (r *impactParticleRenderer) drawBatch(screen *ebiten.Image, particles []impactParticle) {
	vertexCount := len(particles) * 4
	indexCount := len(particles) * 6
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

	for i, p := range particles {
		vi := i * 4
		ii := i * 6
		cx := float32(p.x)
		cy := float32(p.y)
		half := float32(p.radius)
		lifeRate := float32(mathutil.Clamp(float64(p.life)/float64(max(1, p.maxLife)), 0, 1))
		turb := float32(p.turb)
		coreRatio := float32(p.coreRatio)

		r.vertices[vi+0] = impactParticleVertex(cx-half, cy-half, -1, -1, p.col, lifeRate, turb, coreRatio)
		r.vertices[vi+1] = impactParticleVertex(cx+half, cy-half, +1, -1, p.col, lifeRate, turb, coreRatio)
		r.vertices[vi+2] = impactParticleVertex(cx-half, cy+half, -1, +1, p.col, lifeRate, turb, coreRatio)
		r.vertices[vi+3] = impactParticleVertex(cx+half, cy+half, +1, +1, p.col, lifeRate, turb, coreRatio)

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

func impactParticleVertex(dstX, dstY, srcX, srcY float32, col [4]float32, life, turb, coreRatio float32) ebiten.Vertex {
	return ebiten.Vertex{
		DstX:    dstX,
		DstY:    dstY,
		SrcX:    srcX,
		SrcY:    srcY,
		ColorR:  col[0],
		ColorG:  col[1],
		ColorB:  col[2],
		ColorA:  col[3],
		Custom0: life,
		Custom1: turb,
		Custom2: coreRatio,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
