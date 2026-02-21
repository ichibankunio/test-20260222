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
	playerY        float64
	playerVY       float64
	cameraX        float64
	distance       float64
	distanceScore  float64
	bonusScore     int
	obstacles      []obstacle
	collectibles   []collectible
	nextSpawnX     float64
	spawnSerial    int
	combo          int
	maxCombo       int
	lives          int
	invulnFrames   int
	jumpBuffer     int
	coyoteFrames   int
	ticks          int
	gameOver       bool
	gameOverFrames int
}

type obstacle struct {
	x       float64
	width   float64
	height  float64
	bobAmp  float64
	bobFreq float64
	passed  bool
}

type collectible struct {
	x         float64
	y         float64
	radius    float64
	collected bool
}

const (
	groundH             = 360.0
	playerWidth         = 72.0
	playerHeight        = 96.0
	autoScrollBaseSpeed = 4.2
	autoScrollMaxBonus  = 5.2
	gravity             = 1.06
	jumpVelocity        = -27.4
	jumpHoldGravityMul  = 0.62
	maxFallSpeed        = 32.0
	groundTileW         = 96.0
	playerScreenOffsetX = 176.0
	spawnGapMin         = 290.0
	spawnGapMax         = 510.0
	invulnDuration      = 90
)

func (s *MainScene) Init(_ *flib.Game) {
	s.reset()
}

func (s *MainScene) Start(_ *flib.Game) {
	s.reset()
}

func (s *MainScene) Update(g *flib.Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit")
	}

	EnsureBGMPlaying()
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		s.reset()
	}

	s.ticks++
	if s.invulnFrames > 0 {
		s.invulnFrames--
	}

	if s.gameOver {
		s.gameOverFrames--
		if s.gameOverFrames <= 0 {
			result := RunResult{
				Score:       s.score(),
				Distance:    int(s.distance),
				MaxCombo:    s.maxCombo,
				SurviveTick: s.ticks,
			}
			best := updateBestScore(g, result.Score)
			result.BestScore = best
			setLastRun(g, result)
			flib.ShiftSceneWithFadeInOut(g, SceneResult, 28)
		}
		return nil
	}

	speed := s.currentSpeed()
	s.cameraX += speed
	s.distance += speed
	s.distanceScore += speed * 0.55

	onGround := s.playerY >= groundTop()-playerHeight
	if onGround {
		s.coyoteFrames = 7
	} else if s.coyoteFrames > 0 {
		s.coyoteFrames--
	}

	if isJumpInputJustPressed() {
		s.jumpBuffer = 8
	}
	if s.jumpBuffer > 0 {
		s.jumpBuffer--
	}
	if s.jumpBuffer > 0 && s.coyoteFrames > 0 {
		s.playerVY = jumpVelocity
		s.jumpBuffer = 0
		s.coyoteFrames = 0
		if se := FirstSE(); len(se) > 0 {
			PlaySE(se)
		}
	}

	gScale := 1.0
	if isJumpInputPressed() && s.playerVY < 0 {
		gScale = jumpHoldGravityMul
	}
	s.playerVY += gravity * gScale
	if s.playerVY > maxFallSpeed {
		s.playerVY = maxFallSpeed
	}
	s.playerY += s.playerVY

	floorY := groundTop() - playerHeight
	if s.playerY > floorY {
		s.playerY = floorY
		s.playerVY = 0
	}

	s.spawnEntities()
	s.cleanupEntities()
	s.applyScoreFromPassingObstacles()
	s.collectCollectibles()
	s.resolveObstacleHit()

	return nil
}

func (s *MainScene) Draw(screen *ebiten.Image) {
	drawSky(screen, s.cameraX, s.ticks)
	drawHills(screen, s.cameraX)
	drawCitySilhouette(screen, s.cameraX)
	drawGround(screen, s.cameraX)
	drawCollectibles(screen, s.cameraX, s.collectibles, s.ticks)
	drawObstacles(screen, s.cameraX, s.obstacles, s.ticks)
	drawPlayer(screen, playerScreenOffsetX, s.playerY, s.playerVY, s.invulnFrames > 0, s.ticks)
	drawHUD(screen, s)

	if s.gameOver {
		drawOverlay(screen, color.RGBA{0, 0, 0, 150})
		ebitenutil.DebugPrintAt(screen, "CRASHED", 20, 170)
		ebitenutil.DebugPrintAt(screen, "Switching to result...", 20, 192)
	}

	ebitenutil.DebugPrintAt(screen, "SPACE/TAP: JUMP (HOLD to float)", 20, ScreenHeight-96)
	ebitenutil.DebugPrintAt(screen, "R: RETRY  ESC: EXIT", 20, ScreenHeight-74)
}

func (s *MainScene) reset() {
	s.cameraX = 0
	s.distance = 0
	s.distanceScore = 0
	s.bonusScore = 0
	s.playerY = groundTop() - playerHeight
	s.playerVY = 0
	s.obstacles = s.obstacles[:0]
	s.collectibles = s.collectibles[:0]
	s.nextSpawnX = ScreenWidth + 260
	s.spawnSerial = 0
	s.combo = 0
	s.maxCombo = 0
	s.lives = 3
	s.invulnFrames = 0
	s.jumpBuffer = 0
	s.coyoteFrames = 0
	s.ticks = 0
	s.gameOver = false
	s.gameOverFrames = 0
	s.spawnEntities()
}

func (s *MainScene) currentSpeed() float64 {
	bonus := s.distance / 2000
	if bonus > autoScrollMaxBonus {
		bonus = autoScrollMaxBonus
	}
	return autoScrollBaseSpeed + bonus
}

func (s *MainScene) spawnEntities() {
	spawnLimit := s.cameraX + ScreenWidth*1.9
	for s.nextSpawnX < spawnLimit {
		index := s.spawnSerial
		style := index % 6

		s.obstacles = append(s.obstacles, obstacle{
			x:       s.nextSpawnX,
			width:   obstacleWidthFromStyle(style),
			height:  obstacleHeightFromStyle(style, index),
			bobAmp:  obstacleBobAmpFromStyle(style),
			bobFreq: obstacleBobFreqFromStyle(style),
		})

		if index%2 == 1 {
			s.collectibles = append(s.collectibles, collectible{
				x:      s.nextSpawnX + 88,
				y:      collectibleYFromIndex(index),
				radius: 20,
			})
		}
		if index%5 == 4 {
			s.collectibles = append(s.collectibles, collectible{
				x:      s.nextSpawnX - 74,
				y:      collectibleYFromIndex(index + 2),
				radius: 16,
			})
		}

		s.spawnSerial++
		gap := spawnGapFromIndex(s.spawnSerial)
		s.nextSpawnX += gap
	}
}

func obstacleWidthFromStyle(style int) float64 {
	widths := []float64{62, 74, 86, 98, 74, 64}
	return widths[style%len(widths)]
}

func obstacleHeightFromStyle(style, index int) float64 {
	heights := []float64{96, 140, 180, 118, 165, 210}
	h := heights[style%len(heights)]
	if index%7 == 3 {
		h += 22
	}
	return h
}

func obstacleBobAmpFromStyle(style int) float64 {
	if style == 2 || style == 5 {
		return 15
	}
	return 0
}

func obstacleBobFreqFromStyle(style int) float64 {
	if style == 2 {
		return 0.06
	}
	if style == 5 {
		return 0.045
	}
	return 0
}

func collectibleYFromIndex(i int) float64 {
	y := []float64{groundTop() - 220, groundTop() - 300, groundTop() - 380, groundTop() - 260}
	return y[i%len(y)]
}

func spawnGapFromIndex(i int) float64 {
	gaps := []float64{340, 440, 370, 480, 320, 520, 360}
	gap := gaps[i%len(gaps)]
	if gap < spawnGapMin {
		gap = spawnGapMin
	}
	if gap > spawnGapMax {
		gap = spawnGapMax
	}
	return gap
}

func (s *MainScene) cleanupEntities() {
	keepObsFrom := 0
	for keepObsFrom < len(s.obstacles) {
		if s.obstacles[keepObsFrom].x+s.obstacles[keepObsFrom].width >= s.cameraX-220 {
			break
		}
		keepObsFrom++
	}
	if keepObsFrom > 0 {
		copy(s.obstacles, s.obstacles[keepObsFrom:])
		s.obstacles = s.obstacles[:len(s.obstacles)-keepObsFrom]
	}

	keepColFrom := 0
	for keepColFrom < len(s.collectibles) {
		if s.collectibles[keepColFrom].x+s.collectibles[keepColFrom].radius >= s.cameraX-220 {
			break
		}
		keepColFrom++
	}
	if keepColFrom > 0 {
		copy(s.collectibles, s.collectibles[keepColFrom:])
		s.collectibles = s.collectibles[:len(s.collectibles)-keepColFrom]
	}
}

func (s *MainScene) applyScoreFromPassingObstacles() {
	playerX := s.cameraX + playerScreenOffsetX
	for i := range s.obstacles {
		o := &s.obstacles[i]
		if o.passed || o.x+o.width >= playerX {
			continue
		}
		o.passed = true
		s.combo++
		if s.combo > s.maxCombo {
			s.maxCombo = s.combo
		}
		s.bonusScore += 50 + s.combo*9
	}
}

func (s *MainScene) collectCollectibles() {
	playerLeft := s.cameraX + playerScreenOffsetX
	playerRight := playerLeft + playerWidth
	playerTop := s.playerY
	playerBottom := s.playerY + playerHeight

	for i := range s.collectibles {
		c := &s.collectibles[i]
		if c.collected {
			continue
		}
		closestX := clamp(c.x, playerLeft, playerRight)
		closestY := clamp(c.y, playerTop, playerBottom)
		dx := c.x - closestX
		dy := c.y - closestY
		if dx*dx+dy*dy <= c.radius*c.radius {
			c.collected = true
			s.bonusScore += 120 + s.combo*14
		}
	}
}

func (s *MainScene) resolveObstacleHit() {
	if s.invulnFrames > 0 {
		return
	}

	playerLeft := s.cameraX + playerScreenOffsetX
	playerRight := playerLeft + playerWidth
	playerTop := s.playerY
	playerBottom := s.playerY + playerHeight

	for _, o := range s.obstacles {
		obsTop := groundTop() - o.height
		if o.bobAmp > 0 {
			obsTop -= math.Sin(float64(s.ticks)*o.bobFreq) * o.bobAmp
		}
		obsBottom := groundTop()
		obsLeft := o.x
		obsRight := o.x + o.width
		if playerRight <= obsLeft || playerLeft >= obsRight || playerBottom <= obsTop || playerTop >= obsBottom {
			continue
		}

		s.lives--
		s.combo = 0
		s.invulnFrames = invulnDuration
		if se := FirstSE(); len(se) > 0 {
			PlaySE(se)
		}
		if s.lives <= 0 {
			s.gameOver = true
			s.gameOverFrames = 80
		}
		return
	}
}

func drawSky(screen *ebiten.Image, cameraX float64, ticks int) {
	screen.Fill(color.RGBA{R: 114, G: 169, B: 255, A: 255})
	drawFilledRect(screen, 0, 0, ScreenWidth, 420, color.RGBA{R: 148, G: 197, B: 255, A: 190})
	drawFilledRect(screen, 0, 420, ScreenWidth, 360, color.RGBA{R: 205, G: 226, B: 255, A: 120})
	const cloudSpacing = 390.0
	cloudCameraX := cameraX * 0.32
	start := -math.Mod(cloudCameraX, cloudSpacing)
	for x := start - cloudSpacing; x < ScreenWidth+cloudSpacing; x += cloudSpacing {
		sway := math.Sin((x+float64(ticks))*0.008) * 5
		screenX := x + 34
		drawFilledRect(screen, screenX, 154+sway, 210, 58, color.RGBA{255, 255, 255, 230})
		drawFilledRect(screen, screenX+24, 128+sway, 134, 40, color.RGBA{255, 255, 255, 255})
	}
}

func drawHills(screen *ebiten.Image, cameraX float64) {
	const hillW = 420.0
	start := -math.Mod(cameraX*0.58, hillW)
	for x := start - hillW; x < ScreenWidth+hillW; x += hillW {
		drawFilledRect(screen, x, groundTop()-236, hillW, 236, color.RGBA{R: 96, G: 145, B: 107, A: 255})
		drawFilledRect(screen, x+24, groundTop()-182, hillW-80, 182, color.RGBA{R: 76, G: 121, B: 92, A: 255})
	}
}

func drawCitySilhouette(screen *ebiten.Image, cameraX float64) {
	const buildingW = 94.0
	start := -math.Mod(cameraX*0.74, buildingW)
	for x := start - buildingW; x < ScreenWidth+buildingW; x += buildingW {
		i := int((cameraX*0.74 + x) / buildingW)
		h := []float64{220, 290, 180, 260, 210}[abs(i)%5]
		drawFilledRect(screen, x, groundTop()-h, buildingW-12, h, color.RGBA{R: 56, G: 84, B: 120, A: 240})
	}
}

func drawGround(screen *ebiten.Image, cameraX float64) {
	drawFilledRect(screen, 0, groundTop()-20, ScreenWidth, 20, color.RGBA{R: 220, G: 160, B: 102, A: 255})
	start := -math.Mod(cameraX, groundTileW)
	for x := start - groundTileW; x < ScreenWidth+groundTileW; x += groundTileW {
		tile := int((cameraX + x) / groundTileW)
		fill := color.RGBA{R: 189, G: 112, B: 56, A: 255}
		if tile%2 == 0 {
			fill = color.RGBA{R: 173, G: 101, B: 50, A: 255}
		}
		drawFilledRect(screen, x, groundTop(), groundTileW, groundH, fill)
		drawFilledRect(screen, x+8, groundTop()+12, 16, 16, color.RGBA{R: 226, G: 181, B: 129, A: 255})
	}
}

func drawPlayer(screen *ebiten.Image, x, y, vy float64, invuln bool, ticks int) {
	if invuln && ticks%6 < 3 {
		return
	}
	lean := clamp(vy*0.4, -8, 10)
	shadowW := playerWidth + math.Abs(lean)
	drawFilledRect(screen, x-lean*0.2, y+playerHeight+8, shadowW, 10, color.RGBA{0, 0, 0, 70})
	drawFilledRect(screen, x, y, playerWidth, playerHeight, color.RGBA{R: 232, G: 89, B: 66, A: 255})
	drawFilledRect(screen, x, y, playerWidth, 24, color.RGBA{R: 187, G: 34, B: 30, A: 255})
	drawFilledRect(screen, x+18, y+26, 34, 36, color.RGBA{R: 255, G: 219, B: 178, A: 255})
	drawFilledRect(screen, x+8+lean*0.15, y+66, 20, 30, color.RGBA{R: 95, G: 118, B: 210, A: 255})
	drawFilledRect(screen, x+42+lean*0.15, y+66, 20, 30, color.RGBA{R: 95, G: 118, B: 210, A: 255})
}

func drawObstacles(screen *ebiten.Image, cameraX float64, obstacles []obstacle, ticks int) {
	for _, o := range obstacles {
		x := o.x - cameraX
		y := groundTop() - o.height
		if o.bobAmp > 0 {
			y -= math.Sin(float64(ticks)*o.bobFreq) * o.bobAmp
		}
		drawFilledRect(screen, x, y, o.width, o.height, color.RGBA{R: 42, G: 62, B: 91, A: 255})
		drawFilledRect(screen, x+6, y+8, o.width-12, 16, color.RGBA{R: 89, G: 114, B: 149, A: 255})
	}
}

func drawCollectibles(screen *ebiten.Image, cameraX float64, collectibles []collectible, ticks int) {
	for _, c := range collectibles {
		if c.collected {
			continue
		}
		x := c.x - cameraX
		pulse := math.Sin(float64(ticks)*0.15+x*0.03) * 2
		r := c.radius + pulse
		vector.DrawFilledCircle(screen, float32(x), float32(c.y), float32(r), color.RGBA{R: 254, G: 236, B: 95, A: 255}, true)
		vector.StrokeCircle(screen, float32(x), float32(c.y), float32(r+4), 2, color.RGBA{R: 255, G: 252, B: 216, A: 180}, true)
	}
}

func drawHUD(screen *ebiten.Image, s *MainScene) {
	drawFilledRect(screen, 16, 16, ScreenWidth-32, 124, color.RGBA{R: 11, G: 26, B: 47, A: 160})
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SCORE  %06d", s.score()), 28, 30)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DIST   %dm", int(s.distance/11.8)), 28, 54)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("COMBO  x%d", max(1, s.combo)), 28, 78)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SPEED  %.1f", s.currentSpeed()), 28, 102)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("LIFE %s", heartsText(s.lives)), ScreenWidth-220, 78)
}

func heartsText(lives int) string {
	if lives <= 0 {
		return "-"
	}
	t := ""
	for i := 0; i < lives; i++ {
		t += "*"
	}
	return t
}

func drawOverlay(screen *ebiten.Image, c color.RGBA) {
	drawFilledRect(screen, 0, 0, ScreenWidth, ScreenHeight, c)
}

func (s *MainScene) score() int {
	return int(s.distanceScore) + s.bonusScore
}

func groundTop() float64 {
	return ScreenHeight - groundH
}

func drawFilledRect(screen *ebiten.Image, x, y, width, height float64, c color.Color) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(height), c, true)
}

func isJumpInputJustPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	return len(inpututil.AppendJustPressedTouchIDs(nil)) > 0
}

func isJumpInputPressed() bool {
	if ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		return true
	}
	return len(ebiten.AppendTouchIDs(nil)) > 0
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

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *MainScene) GetStatus() int { return 0 }

func (s *MainScene) GetID() flib.SceneID { return SceneMain }
