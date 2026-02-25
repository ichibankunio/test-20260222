package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Danmaku interface {
	Reset()
	Update(playerX, playerY, playerRadius float64, invincible bool) DanmakuTick
	Draw(screen *ebiten.Image)
}

type ImpactPoint struct {
	X float64
	Y float64
}

type DanmakuTick struct {
	Hit            bool
	GaugeGain      float64
	CoinCollecteds []ImpactPoint
	BulletImpacts  []ImpactPoint
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
	frame             int
	bullets           []projectile
	coins             []projectile
	bulletImpactsBuf  []ImpactPoint
	coinCollectedsBuf []ImpactPoint
	spawn             func(*stageDanmaku, int, float64, float64)
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
	d.bulletImpactsBuf = d.bulletImpactsBuf[:0]
	d.coinCollectedsBuf = d.coinCollectedsBuf[:0]
}

func (d *stageDanmaku) Update(playerX, playerY, playerRadius float64, invincible bool) DanmakuTick {
	if d.spawn != nil {
		d.spawn(d, d.frame, playerX, playerY)
	}
	d.moveProjectiles(playerX, playerY)
	d.cleanupProjectiles()
	hit, bulletImpacts := d.hitEnemyBullet(playerX, playerY, playerRadius, invincible)
	gaugeGain, coinCollecteds := d.collectCoins(playerX, playerY, playerRadius)
	d.frame++
	return DanmakuTick{Hit: hit, GaugeGain: gaugeGain, CoinCollecteds: coinCollecteds, BulletImpacts: bulletImpacts}
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
	step := 360.0 / float64(count)
	for i := range count {
		rad := (offsetDeg + float64(i)*step) * math.Pi / 180
		d.bullets = append(d.bullets, projectile{
			x:      x,
			y:      y,
			vx:     math.Cos(rad) * speed,
			vy:     math.Sin(rad) * speed,
			radius: 4.0,
		})
	}
}

func (d *stageDanmaku) spawnAimedSpread(x, y, speed float64, offsets []float64, playerX, playerY float64) {
	base := math.Atan2(playerY-y, playerX-x)
	for _, off := range offsets {
		rad := base + off*math.Pi/180
		d.bullets = append(d.bullets, projectile{
			x:      x,
			y:      y,
			vx:     math.Cos(rad) * speed,
			vy:     math.Sin(rad) * speed,
			radius: 4.0,
		})
	}
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
	d.coins = append(d.coins, projectile{x: x, y: y, vx: vx, vy: vy, radius: 3.2, value: value})
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

func (d *stageDanmaku) hitEnemyBullet(playerX, playerY, playerRadius float64, invincible bool) (bool, []ImpactPoint) {
	n := 0
	hit := false
	impacts := d.bulletImpactsBuf[:0]
	for _, b := range d.bullets {
		if circlesOverlap(playerX, playerY, playerRadius, b.x, b.y, b.radius) {
			hit = true
			if invincible {
				impacts = append(impacts, ImpactPoint{X: b.x, Y: b.y})
				continue
			}
		}
		d.bullets[n] = b
		n++
	}
	if invincible {
		d.bullets = d.bullets[:n]
	}
	d.bulletImpactsBuf = impacts
	return hit, impacts
}

func (d *stageDanmaku) collectCoins(playerX, playerY, playerRadius float64) (float64, []ImpactPoint) {
	n := 0
	gaugeGain := 0.0
	coinCollecteds := d.coinCollectedsBuf[:0]
	collected := false
	for _, c := range d.coins {
		if circlesOverlap(playerX, playerY, playerRadius, c.x, c.y, c.radius) {
			gaugeGain += c.value
			coinCollecteds = append(coinCollecteds, ImpactPoint{X: c.x, Y: c.y})
			collected = true
			continue
		}
		d.coins[n] = c
		n++
	}
	d.coins = d.coins[:n]
	d.coinCollectedsBuf = coinCollecteds
	if collected {
		if se := FirstSE(); len(se) > 0 {
			PlaySE(se)
		}
	}
	return gaugeGain, coinCollecteds
}
