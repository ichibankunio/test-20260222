package game

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

type portraitPixel struct {
	x      float64
	y      float64
	col    color.RGBA
	filled bool
}

type portraitShard struct {
	x           float64
	y           float64
	vx          float64
	vy          float64
	col         color.RGBA
	life        int
	targetIndex int
}

type portraitBuilder struct {
	pixels         []portraitPixel
	unfilled       []int
	shards         []portraitShard
	progressFilled int
}

var (
	portraitHairDark  = color.RGBA{R: 24, G: 28, B: 45, A: 220}
	portraitHairLight = color.RGBA{R: 54, G: 62, B: 96, A: 220}
	portraitSkin      = color.RGBA{R: 255, G: 215, B: 186, A: 222}
	portraitShadow    = color.RGBA{R: 192, G: 140, B: 128, A: 210}
	portraitEye       = color.RGBA{R: 232, G: 242, B: 255, A: 230}
	portraitSuit      = color.RGBA{R: 22, G: 34, B: 56, A: 208}
	portraitTie       = color.RGBA{R: 206, G: 74, B: 110, A: 220}
)

const (
	shardsPerCoin = 20
	maxShardCount = 420
)

func newPortraitBuilder() portraitBuilder {
	p := portraitBuilder{}
	pixels := make([]portraitPixel, 0, 760)
	appendEllipse := func(cx, cy, rx, ry float64, c color.RGBA) {
		for y := -ry; y <= ry; y += 2 {
			for x := -rx; x <= rx; x += 2 {
				nx := x / rx
				ny := y / ry
				if nx*nx+ny*ny <= 1 {
					pixels = append(pixels, portraitPixel{x: cx + x, y: cy + y, col: c})
				}
			}
		}
	}
	appendRect := func(x, y, w, h float64, c color.RGBA) {
		for py := 0.0; py < h; py += 2 {
			for px := 0.0; px < w; px += 2 {
				pixels = append(pixels, portraitPixel{x: x + px, y: y + py, col: c})
			}
		}
	}

	cx := ScreenWidth * 0.5
	appendEllipse(cx, 114, 25, 32, portraitHairDark)
	appendEllipse(cx, 110, 19, 24, portraitHairLight)
	appendEllipse(cx, 114, 16, 22, portraitSkin)
	appendEllipse(cx-8, 117, 6, 9, portraitSkin)
	appendEllipse(cx+8, 117, 6, 9, portraitSkin)
	appendEllipse(cx, 126, 11, 14, portraitShadow)
	appendEllipse(cx-6, 108, 4, 2, portraitEye)
	appendEllipse(cx+6, 108, 4, 2, portraitEye)
	appendRect(cx-5, 136, 10, 10, portraitSkin)
	appendRect(cx-18, 146, 36, 4, portraitSuit)
	appendRect(cx-24, 150, 48, 16, portraitSuit)
	appendRect(cx-3, 146, 6, 20, portraitTie)

	p.pixels = pixels
	p.reset()
	return p
}

func (p *portraitBuilder) reset() {
	p.shards = p.shards[:0]
	p.progressFilled = 0
	p.unfilled = p.unfilled[:0]
	for i := range p.pixels {
		p.pixels[i].filled = false
		p.unfilled = append(p.unfilled, i)
	}
}

func (p *portraitBuilder) onCoinCollected(x, y float64) {
	if len(p.unfilled) == 0 {
		return
	}
	for i := 0; i < shardsPerCoin; i++ {
		if len(p.unfilled) == 0 {
			return
		}
		targetPoolIdx := rand.IntN(len(p.unfilled))
		targetIdx := p.unfilled[targetPoolIdx]
		ang := rand.Float64() * 2 * math.Pi
		spd := 0.35 + rand.Float64()*1.9
		shard := portraitShard{
			x:           x,
			y:           y,
			vx:          math.Cos(ang) * spd,
			vy:          math.Sin(ang)*spd - 0.2,
			col:         p.pixels[targetIdx].col,
			life:        110 + rand.IntN(40),
			targetIndex: targetIdx,
		}
		p.shards = append(p.shards, shard)
	}
	if len(p.shards) > maxShardCount {
		p.shards = p.shards[len(p.shards)-maxShardCount:]
	}
}

func (p *portraitBuilder) update() {
	n := 0
	for i := range p.shards {
		shard := p.shards[i]
		shard.life--
		if shard.life <= 0 {
			continue
		}
		if shard.targetIndex >= 0 && shard.targetIndex < len(p.pixels) && !p.pixels[shard.targetIndex].filled {
			target := p.pixels[shard.targetIndex]
			dx := target.x - shard.x
			dy := target.y - shard.y
			shard.vx = lerp(shard.vx, dx*0.12, 0.08)
			shard.vy = lerp(shard.vy, dy*0.12, 0.08)
			if dx*dx+dy*dy < 1.8*1.8 || shard.life <= 6 {
				p.fill(shard.targetIndex)
				continue
			}
		}
		shard.vx *= 0.985
		shard.vy = shard.vy*0.982 + 0.016
		shard.x += shard.vx
		shard.y += shard.vy
		p.shards[n] = shard
		n++
	}
	p.shards = p.shards[:n]
}

func (p *portraitBuilder) fill(idx int) {
	if idx < 0 || idx >= len(p.pixels) || p.pixels[idx].filled {
		return
	}
	p.pixels[idx].filled = true
	p.progressFilled++
	for i := range p.unfilled {
		if p.unfilled[i] != idx {
			continue
		}
		last := len(p.unfilled) - 1
		p.unfilled[i] = p.unfilled[last]
		p.unfilled = p.unfilled[:last]
		return
	}
}

func (p *portraitBuilder) draw(screen *ebiten.Image) {
	for _, px := range p.pixels {
		if !px.filled {
			continue
		}
		drawFilledRect(screen, px.x, px.y, 2, 2, px.col)
	}
	for _, shard := range p.shards {
		drawFilledRect(screen, shard.x, shard.y, 1.8, 1.8, shard.col)
	}
}

func (p *portraitBuilder) completionRate() float64 {
	if len(p.pixels) == 0 {
		return 1
	}
	return clamp(float64(p.progressFilled)/float64(len(p.pixels)), 0, 1)
}
