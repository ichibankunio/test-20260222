package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/ichibankunio/flib"
)

type MainScene struct {
	playerX      float64
	playerY      float64
	playerRadius float64
	stage        int
	stageFrame   int
	gauge        float64
	gameOver     bool
	bullets      []projectile
	coins        []projectile
	lastTouchID  ebiten.TouchID
	touchActive  bool
	lastTouchX   float64
	lastTouchY   float64
	mouseActive  bool
	lastMouseX   float64
	lastMouseY   float64
}

type projectile struct {
	x      float64
	y      float64
	vx     float64
	vy     float64
	radius float64
	value  float64
}

func (s *MainScene) Init(_ *flib.Game) {
	s.reset()
	if bgm := FirstBGM(); bgm != nil {
		bgm.SetVolume(0.3)
		bgm.Play()
	}
}

func (s *MainScene) Start(_ *flib.Game) {}

func (s *MainScene) Update(_ *flib.Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit")
	}

	if s.gameOver {
		if isRestartInputJustPressed() {
			s.reset()
		}
		return nil
	}

	s.updatePlayerFromSwipe()
	s.spawnStagePattern()
	s.moveProjectiles()
	s.cleanupProjectiles()
	if s.hitEnemyBullet() {
		s.gameOver = true
		if se := FirstSE(); len(se) > 0 {
			PlaySE(se)
		}
		return nil
	}
	s.collectCoins()
	if s.gauge >= stageGaugeMax {
		s.stage++
		s.startStage()
	}
	s.stageFrame++
	return nil
}

func (s *MainScene) Draw(screen *ebiten.Image) {
	screen.Fill(bgNight)
	drawBackdrop(screen)
	drawGauge(screen, s.gauge/stageGaugeMax)
	drawPlayer(screen, s.playerX, s.playerY, s.playerRadius)
	drawBullets(screen, s.bullets)
	drawCoins(screen, s.coins)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("STAGE %d", s.stage), 4, 4)
	ebitenutil.DebugPrintAt(screen, "SWIPE/DRAG: MOVE", 4, 16)
	ebitenutil.DebugPrintAt(screen, "ESC: EXIT", 4, 28)
	if s.gameOver {
		ebitenutil.DebugPrintAt(screen, "GAME OVER", 42, 114)
		ebitenutil.DebugPrintAt(screen, "TAP/SPACE: RETRY", 26, 128)
	}
}

var (
	bgNight       = color.RGBA{R: 7, G: 8, B: 12, A: 255}
	gridDark      = color.RGBA{R: 20, G: 24, B: 32, A: 255}
	uiBorder      = color.RGBA{R: 240, G: 240, B: 240, A: 255}
	uiGaugeFill   = color.RGBA{R: 0, G: 214, B: 172, A: 255}
	playerMain    = color.RGBA{R: 158, G: 37, B: 255, A: 255}
	playerAccent  = color.RGBA{R: 250, G: 250, B: 250, A: 255}
	enemyBullet   = color.RGBA{R: 16, G: 186, B: 166, A: 255}
	enemyBulletIn = color.RGBA{R: 175, G: 255, B: 245, A: 255}
	coinMain      = color.RGBA{R: 255, G: 73, B: 73, A: 255}
	coinAccent    = color.RGBA{R: 255, G: 195, B: 80, A: 255}
)

const (
	playerMoveMargin = 10.0
	stageGaugeMax    = 100.0
)

func (s *MainScene) reset() {
	s.stage = 1
	s.playerX = ScreenWidth / 2
	s.playerY = ScreenHeight - 36
	s.playerRadius = 6
	s.gameOver = false
	s.touchActive = false
	s.mouseActive = false
	s.bullets = s.bullets[:0]
	s.coins = s.coins[:0]
	s.gauge = 0
	s.startStage()
}

func (s *MainScene) startStage() {
	s.stageFrame = 0
	s.gauge = 0
	s.bullets = s.bullets[:0]
	s.coins = s.coins[:0]
}

func (s *MainScene) updatePlayerFromSwipe() {
	touchIDs := ebiten.AppendTouchIDs(nil)
	if len(touchIDs) > 0 {
		id := touchIDs[0]
		x, y := ebiten.TouchPosition(id)
		tx, ty := s.toGamePosition(x, y)
		if !s.touchActive || s.lastTouchID != id {
			s.touchActive = true
			s.lastTouchID = id
			s.lastTouchX = tx
			s.lastTouchY = ty
			return
		}
		s.playerX += tx - s.lastTouchX
		s.playerY += ty - s.lastTouchY
		s.lastTouchX = tx
		s.lastTouchY = ty
		s.clampPlayer()
		return
	}
	s.touchActive = false

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		mx, my := s.toGamePosition(x, y)
		if !s.mouseActive {
			s.mouseActive = true
			s.lastMouseX = mx
			s.lastMouseY = my
			return
		}
		s.playerX += mx - s.lastMouseX
		s.playerY += my - s.lastMouseY
		s.lastMouseX = mx
		s.lastMouseY = my
		s.clampPlayer()
		return
	}
	s.mouseActive = false
}

func (s *MainScene) toGamePosition(x, y int) (float64, float64) {
	w, h := ebiten.WindowSize()
	if w <= 0 || h <= 0 {
		return float64(x), float64(y)
	}
	return float64(x) * ScreenWidth / float64(w), float64(y) * ScreenHeight / float64(h)
}

func (s *MainScene) clampPlayer() {
	left := playerMoveMargin + s.playerRadius
	right := ScreenWidth - playerMoveMargin - s.playerRadius
	top := 44.0
	bottom := ScreenHeight - 14.0 - s.playerRadius
	s.playerX = clamp(s.playerX, left, right)
	s.playerY = clamp(s.playerY, top, bottom)
}

func (s *MainScene) spawnStagePattern() {
	switch (s.stage - 1) % 3 {
	case 0:
		s.spawnStage1()
	case 1:
		s.spawnStage2()
	default:
		s.spawnStage3()
	}
}

func (s *MainScene) spawnStage1() {
	if s.stageFrame%28 == 0 {
		s.spawnSpread(ScreenWidth/2, -8, 1.35, []float64{70, 82, 94, 106, 118})
	}
	if s.stageFrame%54 == 22 {
		s.spawnSpread(-8, 86, 1.10, []float64{18, 30, 42})
		s.spawnSpread(ScreenWidth+8, 126, 1.10, []float64{138, 150, 162})
	}
	if s.stageFrame%66 == 10 {
		xs := []float64{24, 48, 72, 96, 120, 84, 60}
		s.spawnCoin(xs[(s.stageFrame/66)%len(xs)], -6, 0, 0.95, 20)
	}
}

func (s *MainScene) spawnStage2() {
	if s.stageFrame%24 == 0 {
		s.spawnSpread(10, -8, 1.50, []float64{62, 76, 90})
		s.spawnSpread(ScreenWidth-10, -8, 1.50, []float64{90, 104, 118})
	}
	if s.stageFrame%42 == 18 {
		y := 72.0 + float64((s.stageFrame/42)%4)*28.0
		s.spawnSpread(-8, y, 1.22, []float64{15, 27})
		s.spawnSpread(ScreenWidth+8, y+14, 1.22, []float64{153, 165})
	}
	if s.stageFrame%58 == 16 {
		xs := []float64{18, 126, 36, 108, 54, 90, 72}
		s.spawnCoin(xs[(s.stageFrame/58)%len(xs)], -6, 0, 1.05, 20)
	}
}

func (s *MainScene) spawnStage3() {
	if s.stageFrame%18 == 0 {
		start := float64((s.stageFrame / 18 % 10) * 14)
		s.spawnSpread(start+2, -8, 1.65, []float64{74, 90, 106})
		s.spawnSpread(ScreenWidth-start-2, -8, 1.65, []float64{74, 90, 106})
	}
	if s.stageFrame%38 == 8 {
		s.spawnSpread(ScreenWidth/2, 36, 1.35, []float64{188, 202, 216, 324, 338, 352})
	}
	if s.stageFrame%52 == 24 {
		xs := []float64{24, 72, 120, 48, 96, 72}
		s.spawnCoin(xs[(s.stageFrame/52)%len(xs)], -6, 0, 1.15, 20)
	}
}

func (s *MainScene) spawnSpread(x, y, speed float64, angles []float64) {
	for _, deg := range angles {
		rad := deg * math.Pi / 180
		s.bullets = append(s.bullets, projectile{
			x:      x,
			y:      y,
			vx:     math.Cos(rad) * speed,
			vy:     math.Sin(rad) * speed,
			radius: 4.6,
		})
	}
}

func (s *MainScene) spawnCoin(x, y, vx, vy, value float64) {
	s.coins = append(s.coins, projectile{x: x, y: y, vx: vx, vy: vy, radius: 6.0, value: value})
}

func (s *MainScene) moveProjectiles() {
	for i := range s.bullets {
		s.bullets[i].x += s.bullets[i].vx
		s.bullets[i].y += s.bullets[i].vy
	}
	for i := range s.coins {
		s.coins[i].x += s.coins[i].vx
		s.coins[i].y += s.coins[i].vy
	}
}

func (s *MainScene) cleanupProjectiles() {
	s.bullets = keepOnScreen(s.bullets)
	s.coins = keepOnScreen(s.coins)
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

func (s *MainScene) hitEnemyBullet() bool {
	for _, b := range s.bullets {
		if circlesOverlap(s.playerX, s.playerY, s.playerRadius, b.x, b.y, b.radius) {
			return true
		}
	}
	return false
}

func (s *MainScene) collectCoins() {
	n := 0
	for _, c := range s.coins {
		if circlesOverlap(s.playerX, s.playerY, s.playerRadius, c.x, c.y, c.radius) {
			s.gauge = clamp(s.gauge+c.value, 0, stageGaugeMax)
			if se := FirstSE(); len(se) > 0 {
				PlaySE(se)
			}
			continue
		}
		s.coins[n] = c
		n++
	}
	s.coins = s.coins[:n]
}

func drawBackdrop(screen *ebiten.Image) {
	for y := 0.0; y < ScreenHeight; y += 16 {
		for x := 0.0; x < ScreenWidth; x += 16 {
			shade := gridDark
			if int((x+y)/16)%2 == 0 {
				shade = color.RGBA{R: 14, G: 18, B: 26, A: 255}
			}
			drawFilledRect(screen, x, y, 16, 16, shade)
		}
	}
}

func drawGauge(screen *ebiten.Image, rate float64) {
	x := 8.0
	y := 38.0
	w := ScreenWidth - 16.0
	h := 9.0
	drawFilledRect(screen, x-1, y-1, w+2, h+2, uiBorder)
	drawFilledRect(screen, x, y, w, h, color.RGBA{R: 16, G: 22, B: 30, A: 255})
	drawFilledRect(screen, x, y, w*clamp(rate, 0, 1), h, uiGaugeFill)
}

func drawPlayer(screen *ebiten.Image, x, y, r float64) {
	drawFilledRect(screen, x-r, y-r, r*2, r*2, playerMain)
	drawFilledRect(screen, x-r+2, y-r+2, r*2-4, r*2-4, playerAccent)
	drawFilledRect(screen, x-r+3, y-r+3, r*2-6, r*2-6, playerMain)
}

func drawBullets(screen *ebiten.Image, bullets []projectile) {
	for _, b := range bullets {
		vector.DrawFilledCircle(screen, float32(b.x), float32(b.y), float32(b.radius), enemyBullet, true)
		vector.DrawFilledCircle(screen, float32(b.x), float32(b.y), float32(b.radius*0.45), enemyBulletIn, true)
	}
}

func drawCoins(screen *ebiten.Image, coins []projectile) {
	for _, c := range coins {
		vector.DrawFilledCircle(screen, float32(c.x), float32(c.y), float32(c.radius), coinMain, true)
		vector.DrawFilledCircle(screen, float32(c.x), float32(c.y), float32(c.radius*0.55), coinAccent, true)
	}
}

func drawFilledRect(screen *ebiten.Image, x, y, width, height float64, c color.Color) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(height), c, true)
}

func circlesOverlap(ax, ay, ar, bx, by, br float64) bool {
	dx := ax - bx
	dy := ay - by
	distSq := dx*dx + dy*dy
	r := ar + br
	return distSq <= r*r
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func isJumpInputJustPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	return len(inpututil.AppendJustPressedTouchIDs(nil)) > 0
}

func isRestartInputJustPressed() bool {
	return isJumpInputJustPressed()
}

func (s *MainScene) GetStatus() int { return 0 }

func (s *MainScene) GetID() flib.SceneID { return SceneMain }
