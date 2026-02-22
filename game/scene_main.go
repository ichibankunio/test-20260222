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
	gauge        float64
	gameOver     bool
	danmaku      Danmaku
	lastTouchID  ebiten.TouchID
	touchActive  bool
	lastTouchX   float64
	lastTouchY   float64
	mouseActive  bool
	lastMouseX   float64
	lastMouseY   float64
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
			s.retryStage()
		}
		return nil
	}

	s.updatePlayerFromSwipe()
	if s.danmaku != nil {
		tick := s.danmaku.Update(s.playerX, s.playerY, s.playerRadius)
		if tick.Hit {
			s.gameOver = true
			if se := FirstSE(); len(se) > 0 {
				PlaySE(se)
			}
			return nil
		}
		s.gauge = clamp(s.gauge+tick.GaugeGain, 0, stageGaugeMax)
	}
	if s.gauge >= stageGaugeMax {
		s.stage++
		s.startStage()
	}
	return nil
}

func (s *MainScene) Draw(screen *ebiten.Image) {
	screen.Fill(bgNight)
	drawBackdrop(screen)
	drawGauge(screen, s.stage, s.gauge/stageGaugeMax)
	drawPlayer(screen, s.playerX, s.playerY, s.playerRadius)
	if s.danmaku != nil {
		s.danmaku.Draw(screen)
	}

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
	stageCycleCount  = 10
)

func (s *MainScene) reset() {
	s.stage = 1
	s.retryStage()
}

func (s *MainScene) retryStage() {
	s.playerX = ScreenWidth / 2
	s.playerY = ScreenHeight - 36
	s.playerRadius = 6
	s.gameOver = false
	s.touchActive = false
	s.mouseActive = false
	s.gauge = 0
	s.startStage()
}

func (s *MainScene) startStage() {
	s.gauge = 0
	s.danmaku = newDanmakuForStage(s.stage)
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

func drawGauge(screen *ebiten.Image, stage int, rate float64) {
	level := stage
	if level < 1 {
		level = 1
	}
	nextLevel := level + 1

	outerX := 8.0
	outerY := 36.0
	outerW := ScreenWidth - 16.0
	outerH := 18.0
	border := 1.0
	levelBoxSize := outerH - border*2
	outerLevelBoxSize := levelBoxSize + border*2
	nextBoxW := 24.0
	innerX := outerX + border
	innerY := outerY + border
	innerW := outerW - border*2
	innerH := outerH - border*2
	gaugeX := innerX + levelBoxSize
	gaugeW := innerW - levelBoxSize - nextBoxW

	drawGaugeShell(screen, outerX, outerY, outerW, outerH, outerLevelBoxSize, nextBoxW+border*2, 0.56, uiBorder)
	drawGaugeShell(screen, innerX, innerY, innerW, innerH, levelBoxSize, nextBoxW, 0.56, color.RGBA{R: 16, G: 22, B: 30, A: 255})

	fillW := gaugeW * clamp(rate, 0, 1)
	if fillW > 0 {
		pipeH := innerH * 0.56
		pipeY := innerY + (innerH-pipeH)/2
		drawFilledRect(screen, gaugeX, pipeY, fillW, pipeH, coinMain)
	}

	levelText := fmt.Sprintf("%d", level)
	nextLevelText := fmt.Sprintf("%d", nextLevel)
	levelTextW := float64(len(levelText)) * 6.0
	nextLevelTextW := float64(len(nextLevelText)) * 6.0
	levelTextX := int(innerX + (levelBoxSize-levelTextW)/2)
	nextLevelTextX := int(innerX + innerW - nextBoxW + (nextBoxW-nextLevelTextW)/2)
	textY := int(innerY + (innerH-8.0)/2)
	ebitenutil.DebugPrintAt(screen, levelText, levelTextX, textY)
	ebitenutil.DebugPrintAt(screen, nextLevelText, nextLevelTextX, textY)
}

func drawGaugeShell(screen *ebiten.Image, x, y, width, height, levelBoxWidth, nextBoxWidth, centerRate float64, c color.Color) {
	if width <= 0 || height <= 0 {
		return
	}
	levelW := clamp(levelBoxWidth, 0, width)
	if levelW == 0 {
		drawCapsule(screen, x, y, width, height, c)
		return
	}
	nextW := clamp(nextBoxWidth, 0, width-levelW)
	centerW := width - levelW - nextW
	centerH := clamp(height*centerRate, 1, height)
	centerY := y + (height-centerH)/2
	if centerW > centerH && nextW > 0 {
		drawGaugeShellPath(screen, x, y, width, height, levelW, nextW, centerY, centerH, c)
		return
	}

	drawRoundedRect(screen, x, y, levelW, height, 3.0, c)
	if nextW > 0 {
		drawRoundedRect(screen, x+width-nextW, y, nextW, height, 3.0, c)
	}
	if centerW > 0 {
		drawCapsule(screen, x+levelW, centerY, centerW, centerH, c)
	}
}

func drawGaugeShellPath(screen *ebiten.Image, x, y, width, height, levelW, nextW, centerY, centerH float64, c color.Color) {
	left := x
	top := y
	right := x + width
	bottom := y + height
	leftJoin := x + levelW
	rightJoin := right - nextW
	pipeR := centerH / 2
	pipeMidY := centerY + pipeR
	pipeTop := centerY
	pipeBottom := centerY + centerH
	leftR := clamp(3.0, 0, min(levelW, height)/2)
	rightR := clamp(3.0, 0, min(nextW, height)/2)

	var p vector.Path
	p.MoveTo(float32(left+leftR), float32(top))
	p.LineTo(float32(leftJoin), float32(top))
	p.LineTo(float32(leftJoin), float32(pipeMidY))
	p.Arc(float32(leftJoin+pipeR), float32(pipeMidY), float32(pipeR), float32(math.Pi), float32(3*math.Pi/2), vector.CounterClockwise)
	p.LineTo(float32(rightJoin-pipeR), float32(pipeTop))
	p.Arc(float32(rightJoin-pipeR), float32(pipeMidY), float32(pipeR), float32(3*math.Pi/2), 0, vector.CounterClockwise)
	p.LineTo(float32(rightJoin), float32(top))
	p.LineTo(float32(right-rightR), float32(top))
	p.Arc(float32(right-rightR), float32(top+rightR), float32(rightR), float32(3*math.Pi/2), 0, vector.CounterClockwise)
	p.LineTo(float32(right), float32(bottom-rightR))
	p.Arc(float32(right-rightR), float32(bottom-rightR), float32(rightR), 0, float32(math.Pi/2), vector.CounterClockwise)
	p.LineTo(float32(rightJoin), float32(bottom))
	p.LineTo(float32(rightJoin), float32(pipeMidY))
	p.Arc(float32(rightJoin-pipeR), float32(pipeMidY), float32(pipeR), 0, float32(math.Pi/2), vector.CounterClockwise)
	p.LineTo(float32(leftJoin+pipeR), float32(pipeBottom))
	p.Arc(float32(leftJoin+pipeR), float32(pipeMidY), float32(pipeR), float32(math.Pi/2), float32(math.Pi), vector.CounterClockwise)
	p.LineTo(float32(leftJoin), float32(bottom))
	p.LineTo(float32(left+leftR), float32(bottom))
	p.Arc(float32(left+leftR), float32(bottom-leftR), float32(leftR), float32(math.Pi/2), float32(math.Pi), vector.CounterClockwise)
	p.LineTo(float32(left), float32(top+leftR))
	p.Arc(float32(left+leftR), float32(top+leftR), float32(leftR), float32(math.Pi), float32(3*math.Pi/2), vector.CounterClockwise)
	p.Close()

	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vs) == 0 || len(is) == 0 {
		return
	}
	r, g, b, a := c.RGBA()
	fr := float32(r) / 0xffff
	fg := float32(g) / 0xffff
	fb := float32(b) / 0xffff
	fa := float32(a) / 0xffff
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = fr
		vs[i].ColorG = fg
		vs[i].ColorB = fb
		vs[i].ColorA = fa
	}
	screen.DrawTriangles(vs, is, whitePixelImage, nil)
}

func drawCapsule(screen *ebiten.Image, x, y, width, height float64, c color.Color) {
	if width <= 0 || height <= 0 {
		return
	}
	r := height / 2
	if width <= height {
		vector.DrawFilledCircle(screen, float32(x+width/2), float32(y+r), float32(width/2), c, true)
		return
	}
	drawFilledRect(screen, x+r, y, width-r*2, height, c)
	vector.DrawFilledCircle(screen, float32(x+r), float32(y+r), float32(r), c, true)
	vector.DrawFilledCircle(screen, float32(x+width-r), float32(y+r), float32(r), c, true)
}

func drawLeftRoundedRect(screen *ebiten.Image, x, y, width, height float64, c color.Color) {
	if width <= 0 || height <= 0 {
		return
	}
	r := height / 2
	if width <= r {
		drawFilledRect(screen, x, y, width, height, c)
		return
	}
	drawFilledRect(screen, x+r, y, width-r, height, c)
	vector.DrawFilledCircle(screen, float32(x+r), float32(y+r), float32(r), c, true)
}

func drawRoundedRect(screen *ebiten.Image, x, y, width, height, radius float64, c color.Color) {
	if width <= 0 || height <= 0 {
		return
	}
	r := clamp(radius, 0, min(width, height)/2)
	if r == 0 {
		drawFilledRect(screen, x, y, width, height, c)
		return
	}
	drawFilledRect(screen, x+r, y, width-r*2, height, c)
	drawFilledRect(screen, x, y+r, r, height-r*2, c)
	drawFilledRect(screen, x+width-r, y+r, r, height-r*2, c)
	vector.DrawFilledCircle(screen, float32(x+r), float32(y+r), float32(r), c, true)
	vector.DrawFilledCircle(screen, float32(x+width-r), float32(y+r), float32(r), c, true)
	vector.DrawFilledCircle(screen, float32(x+r), float32(y+height-r), float32(r), c, true)
	vector.DrawFilledCircle(screen, float32(x+width-r), float32(y+height-r), float32(r), c, true)
}

func drawPlayer(screen *ebiten.Image, x, y, r float64) {
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r), playerMain, true)
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r*0.65), playerAccent, true)
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r*0.4), playerMain, true)
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

var whitePixelImage = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img
}()

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

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*clamp(t, 0, 1)
}

func stageNumber(stage int) int {
	if stage <= 0 {
		return 0
	}
	return (stage - 1) % stageCycleCount
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
