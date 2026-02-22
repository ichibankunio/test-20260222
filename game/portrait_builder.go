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
	marbleCol   color.RGBA
	marbleDX    float64
	marbleDY    float64
	life        int
	targetIndex int
}

type portraitBuilder struct {
	pixels         []portraitPixel
	unfilled       []int
	shards         []portraitShard
	progressFilled int
}

const (
	shardsPerCoin = 20
	maxShardCount = 420
)

func newPortraitBuilder() portraitBuilder {
	p := portraitBuilder{}
	src := GetImage("bishonen.png")
	if src == nil {
		src = GetImage("zentablue.png")
	}
	if src == nil {
		p.pixels = nil
		p.reset()
		return p
	}

	b := src.Bounds()
	sw := b.Dx()
	sh := b.Dy()
	if sw <= 0 || sh <= 0 {
		p.pixels = nil
		p.reset()
		return p
	}

	maxW := 96.0
	maxH := 160.0
	scale := math.Min(maxW/float64(sw), maxH/float64(sh))
	if scale <= 0 {
		scale = 1
	}
	targetW := float64(sw) * scale
	targetH := float64(sh) * scale
	left := (ScreenWidth - targetW) * 0.5
	top := 70.0
	if top+targetH > ScreenHeight-8 {
		top = ScreenHeight - 8 - targetH
	}

	const pxSize = 2
	targetStep := float64(pxSize)
	capacity := int((targetW / targetStep) * (targetH / targetStep))
	pixels := make([]portraitPixel, 0, capacity)
	for ty := 0.0; ty < targetH; ty += targetStep {
		sy := int(ty / scale)
		if sy < 0 || sy >= sh {
			continue
		}
		for tx := 0.0; tx < targetW; tx += targetStep {
			sx := int(tx / scale)
			if sx < 0 || sx >= sw {
				continue
			}
			col := color.RGBAModel.Convert(src.At(b.Min.X+sx, b.Min.Y+sy)).(color.RGBA)
			if col.A < 8 {
				continue
			}
			pixels = append(pixels, portraitPixel{
				x:   left + tx,
				y:   top + ty,
				col: col,
			})
		}
	}

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
		marbleIdx := rand.IntN(len(p.pixels))
		ang := rand.Float64() * 2 * math.Pi
		spd := 0.35 + rand.Float64()*1.9
		shard := portraitShard{
			x:           x,
			y:           y,
			vx:          math.Cos(ang) * spd,
			vy:          math.Sin(ang)*spd - 0.2,
			col:         p.pixels[targetIdx].col,
			marbleCol:   p.pixels[marbleIdx].col,
			marbleDX:    rand.Float64() * 0.8,
			marbleDY:    rand.Float64() * 0.8,
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
		drawFilledRect(screen, shard.x+shard.marbleDX, shard.y+shard.marbleDY, 0.9, 0.9, shard.marbleCol)
	}
}

func (p *portraitBuilder) completionRate() float64 {
	if len(p.pixels) == 0 {
		return 1
	}
	return clamp(float64(p.progressFilled)/float64(len(p.pixels)), 0, 1)
}
