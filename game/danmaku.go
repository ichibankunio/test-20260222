package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Danmaku interface {
	Reset()
	Update(playerX, playerY, playerRadius float64) DanmakuTick
	Draw(screen *ebiten.Image)
}

type DanmakuTick struct {
	Hit       bool
	GaugeGain float64
}

type projectile struct {
	x      float64
	y      float64
	vx     float64
	vy     float64
	radius float64
	value  float64
	homing float64
	life   int
}

type stageDanmaku struct {
	frame   int
	bullets []projectile
	coins   []projectile
	spawn   func(*stageDanmaku, int, float64, float64)
}

func newStageDanmaku(spawn func(*stageDanmaku, int, float64, float64)) Danmaku {
	d := &stageDanmaku{spawn: spawn}
	d.Reset()
	return d
}

func (d *stageDanmaku) Reset() {
	d.frame = 0
	d.bullets = d.bullets[:0]
	d.coins = d.coins[:0]
}

func (d *stageDanmaku) Update(playerX, playerY, playerRadius float64) DanmakuTick {
	if d.spawn != nil {
		d.spawn(d, d.frame, playerX, playerY)
	}
	d.moveProjectiles(playerX, playerY)
	d.cleanupProjectiles()
	hit := d.hitEnemyBullet(playerX, playerY, playerRadius)
	gaugeGain := d.collectCoins(playerX, playerY, playerRadius)
	d.frame++
	return DanmakuTick{Hit: hit, GaugeGain: gaugeGain}
}

func (d *stageDanmaku) Draw(screen *ebiten.Image) {
	drawBullets(screen, d.bullets)
	drawCoins(screen, d.coins)
}

func (d *stageDanmaku) spawnSpread(x, y, speed float64, angles []float64) {
	for _, deg := range angles {
		rad := deg * math.Pi / 180
		d.bullets = append(d.bullets, projectile{
			x:      x,
			y:      y,
			vx:     math.Cos(rad) * speed,
			vy:     math.Sin(rad) * speed,
			radius: 4.0,
		})
	}
}

func (d *stageDanmaku) spawnRing(x, y, speed float64, count int, offsetDeg float64) {
	if count <= 0 {
		return
	}
	angles := make([]float64, 0, count)
	step := 360.0 / float64(count)
	for i := range count {
		angles = append(angles, offsetDeg+float64(i)*step)
	}
	d.spawnSpread(x, y, speed, angles)
}

func (d *stageDanmaku) spawnAimedSpread(x, y, speed float64, offsets []float64, playerX, playerY float64) {
	base := math.Atan2(playerY-y, playerX-x) * 180 / math.Pi
	angles := make([]float64, 0, len(offsets))
	for _, off := range offsets {
		angles = append(angles, base+off)
	}
	d.spawnSpread(x, y, speed, angles)
}

func (d *stageDanmaku) spawnHoming(x, y, speed, homing float64, life int, playerX, playerY float64) {
	angle := math.Atan2(playerY-y, playerX-x)
	d.bullets = append(d.bullets, projectile{
		x:      x,
		y:      y,
		vx:     math.Cos(angle) * speed,
		vy:     math.Sin(angle) * speed,
		radius: 4.2,
		homing: homing,
		life:   life,
	})
}

func (d *stageDanmaku) spawnCoin(x, y, vx, vy, value float64) {
	d.coins = append(d.coins, projectile{x: x, y: y, vx: vx, vy: vy, radius: 6.0, value: value})
}

func (d *stageDanmaku) moveProjectiles(playerX, playerY float64) {
	for i := range d.bullets {
		if d.bullets[i].homing > 0 && d.bullets[i].life != 0 {
			dx := playerX - d.bullets[i].x
			dy := playerY - d.bullets[i].y
			dist := math.Hypot(dx, dy)
			if dist > 0.001 {
				speed := math.Hypot(d.bullets[i].vx, d.bullets[i].vy)
				targetVX := dx / dist * speed
				targetVY := dy / dist * speed
				d.bullets[i].vx = lerp(d.bullets[i].vx, targetVX, d.bullets[i].homing)
				d.bullets[i].vy = lerp(d.bullets[i].vy, targetVY, d.bullets[i].homing)
			}
			if d.bullets[i].life > 0 {
				d.bullets[i].life--
			}
		}
		d.bullets[i].x += d.bullets[i].vx
		d.bullets[i].y += d.bullets[i].vy
	}
	for i := range d.coins {
		d.coins[i].x += d.coins[i].vx
		d.coins[i].y += d.coins[i].vy
	}
}

func (d *stageDanmaku) cleanupProjectiles() {
	d.bullets = keepOnScreen(d.bullets)
	d.coins = keepOnScreen(d.coins)
}

func keepOnScreen(items []projectile) []projectile {
	n := 0
	for _, item := range items {
		margin := item.radius + 12
		if item.x < -margin || item.x > ScreenWidth+margin || item.y < -margin || item.y > ScreenHeight+margin {
			continue
		}
		items[n] = item
		n++
	}
	return items[:n]
}

func (d *stageDanmaku) hitEnemyBullet(playerX, playerY, playerRadius float64) bool {
	for _, b := range d.bullets {
		if circlesOverlap(playerX, playerY, playerRadius, b.x, b.y, b.radius) {
			return true
		}
	}
	return false
}

func (d *stageDanmaku) collectCoins(playerX, playerY, playerRadius float64) float64 {
	n := 0
	gaugeGain := 0.0
	for _, c := range d.coins {
		if circlesOverlap(playerX, playerY, playerRadius, c.x, c.y, c.radius) {
			gaugeGain += c.value
			if se := FirstSE(); len(se) > 0 {
				PlaySE(se)
			}
			continue
		}
		d.coins[n] = c
		n++
	}
	d.coins = d.coins[:n]
	return gaugeGain
}
