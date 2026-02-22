package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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
	particles    impactParticleSystem
	portrait     portraitBuilder
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
	s.particles.update()
	s.portrait.update()

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
		for _, p := range tick.CoinCollecteds {
			s.particles.spawnCoinPickup(p.X, p.Y)
			s.portrait.onCoinCollected(p.X, p.Y)
		}
		if tick.Hit {
			s.particles.spawnPlayerBurst(s.playerX, s.playerY)
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
	s.portrait.draw(screen)
	drawGauge(screen, s.gauge/stageGaugeMax, s.stage)
	drawPlayer(screen, s.playerX, s.playerY, s.playerRadius)
	if s.danmaku != nil {
		s.danmaku.Draw(screen)
	}
	s.particles.draw(screen)

	drawUITextAt(screen, fmt.Sprintf("STAGE %d", s.stage), 4, 4)
	drawUITextAt(screen, fmt.Sprintf("ART %02d%%", int(s.portrait.completionRate()*100)), 82, 4)
	drawUITextAt(screen, "SWIPE/DRAG: MOVE", 4, 16)
	drawUITextAt(screen, "ESC: EXIT", 4, 28)
	if s.gameOver {
		drawUITextCentered(screen, "GAME OVER", 0, 112, ScreenWidth, 12, 16)
		drawUITextCentered(screen, "TAP/SPACE: RETRY", 0, 128, ScreenWidth, 12, 12)
	}
}

var (
	bgNight       = color.RGBA{R: 7, G: 8, B: 12, A: 255}
	gridDark      = color.RGBA{R: 20, G: 24, B: 32, A: 255}
	uiBorder      = color.RGBA{R: 240, G: 240, B: 240, A: 255}
	uiGaugeFill   = color.RGBA{R: 150, G: 43, B: 196, A: 255}
	uiGaugeBase   = color.RGBA{R: 5, G: 5, B: 8, A: 255}
	playerMain    = color.RGBA{R: 158, G: 37, B: 255, A: 255}
	playerAccent  = color.RGBA{R: 250, G: 250, B: 250, A: 255}
	enemyBullet   = color.RGBA{R: 16, G: 186, B: 166, A: 255}
	enemyBulletIn = color.RGBA{R: 175, G: 255, B: 245, A: 255}
	coinMain      = color.RGBA{R: 255, G: 73, B: 73, A: 255}
	coinAccent    = color.RGBA{R: 255, G: 195, B: 80, A: 255}
	coinPixels    []coinPixel
	coinPXSize    = 2.0
)

type coinPixel struct {
	dx  float64
	dy  float64
	col color.RGBA
}

const (
	playerMoveMargin = 10.0
	stageGaugeMax    = 100.0
	stageCycleCount  = 10
)

func (s *MainScene) reset() {
	s.stage = 1
	s.portrait = newPortraitBuilder()
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
	s.particles.reset()
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

func drawGauge(screen *ebiten.Image, rate float64, stage int) {
	leftX := 8.0
	topY := 38.0
	boxW := 44.0
	boxH := 28.0
	barY := topY + 10.0
	barH := 8.0
	rightX := ScreenWidth - 8.0 - boxW
	barX := leftX + boxW
	barW := rightX - barX

	drawFilledRect(screen, leftX, topY, boxW, boxH, uiGaugeBase)
	drawFilledRect(screen, barX, barY, barW, barH, uiGaugeBase)
	drawFilledRect(screen, rightX, topY, boxW, boxH, uiGaugeBase)

	fillPad := 2.0
	fillW := (boxW - fillPad*2) * clamp(rate, 0, 1)
	drawFilledRect(screen, leftX+fillPad, topY+fillPad, fillW, boxH-fillPad*2, uiGaugeFill)

	// One-stroke style frame: trace the full outer contour and use rounded joins.
	var frame vector.Path
	frame.MoveTo(float32(leftX), float32(topY))
	frame.LineTo(float32(leftX+boxW), float32(topY))
	frame.LineTo(float32(leftX+boxW), float32(barY))
	frame.LineTo(float32(rightX), float32(barY))
	frame.LineTo(float32(rightX), float32(topY))
	frame.LineTo(float32(rightX+boxW), float32(topY))
	frame.LineTo(float32(rightX+boxW), float32(topY+boxH))
	frame.LineTo(float32(rightX), float32(topY+boxH))
	frame.LineTo(float32(rightX), float32(barY+barH))
	frame.LineTo(float32(leftX+boxW), float32(barY+barH))
	frame.LineTo(float32(leftX+boxW), float32(topY+boxH))
	frame.LineTo(float32(leftX), float32(topY+boxH))
	frame.Close()

	frameDrawOp := &vector.DrawPathOptions{}
	frameDrawOp.AntiAlias = true
	frameDrawOp.ColorScale.ScaleWithColor(uiBorder)
	frameStroke := &vector.StrokeOptions{}
	frameStroke.Width = 2
	frameStroke.LineJoin = vector.LineJoinRound
	vector.StrokePath(screen, &frame, frameStroke, frameDrawOp)

	drawUITextCentered(screen, fmt.Sprintf("%d", stage), int(leftX), int(topY), int(boxW), int(boxH), 16)
	drawUITextCentered(screen, fmt.Sprintf("%d", stage+1), int(rightX), int(topY), int(boxW), int(boxH), 16)
}

func drawUITextAt(screen *ebiten.Image, body string, x, y int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(uiBorder)
	text.Draw(screen, body, GetGoTextFace(12), op)
}

func drawUITextCentered(screen *ebiten.Image, body string, x, y, w, h, size int) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+w/2), float64(y+h/2))
	op.LayoutOptions.PrimaryAlign = text.AlignCenter
	op.LayoutOptions.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(uiBorder)
	text.Draw(screen, body, GetGoTextFace(size), op)
}

func drawPlayer(screen *ebiten.Image, x, y, r float64) {
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r), playerMain, true)
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r*0.65), playerAccent, true)
	vector.DrawFilledCircle(screen, float32(x), float32(y), float32(r*0.4), playerMain, true)
}

func drawBullets(screen *ebiten.Image, bullets []projectile) {
	if globalBulletShaderRenderer.draw(screen, bullets) {
		return
	}
	for _, b := range bullets {
		vector.DrawFilledCircle(screen, float32(b.x), float32(b.y), float32(b.radius), enemyBullet, true)
		vector.DrawFilledCircle(screen, float32(b.x), float32(b.y), float32(b.radius*0.45), enemyBulletIn, true)
	}
}

func drawCoins(screen *ebiten.Image, coins []projectile) {
	if len(coinPixels) == 0 {
		coinPixels = buildCoinPixels(6, coinPXSize)
	}
	if len(coinPixels) == 0 {
		for _, c := range coins {
			vector.DrawFilledCircle(screen, float32(c.x), float32(c.y), float32(c.radius), coinMain, true)
			vector.DrawFilledCircle(screen, float32(c.x), float32(c.y), float32(c.radius*0.55), coinAccent, true)
		}
		return
	}
	for _, c := range coins {
		scale := c.radius / 6.0
		size := coinPXSize * scale
		for _, px := range coinPixels {
			drawFilledRect(screen, c.x+px.dx*scale, c.y+px.dy*scale, size, size, px.col)
		}
	}
}

func buildCoinPixels(radius, pxSize float64) []coinPixel {
	src := GetImage("bishonen.png")
	if src == nil {
		src = GetImage("zentablue.png")
	}
	if src == nil || pxSize <= 0 || radius <= 0 {
		return nil
	}
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	pixels := make([]coinPixel, 0, int(math.Pi*radius*radius/(pxSize*pxSize)))
	diameter := radius * 2
	for y := -radius; y < radius; y += pxSize {
		for x := -radius; x < radius; x += pxSize {
			cx := x + pxSize*0.5
			cy := y + pxSize*0.5
			if cx*cx+cy*cy > radius*radius {
				continue
			}

			u := clamp((cx+radius)/diameter, 0, 1)
			v := clamp((cy+radius)/diameter, 0, 1)
			sx := int(u * float64(w-1))
			sy := int(v * float64(h-1))
			col := color.RGBAModel.Convert(src.At(b.Min.X+sx, b.Min.Y+sy)).(color.RGBA)
			if col.A < 8 {
				continue
			}
			pixels = append(pixels, coinPixel{
				dx:  x - pxSize*0.5,
				dy:  y - pxSize*0.5,
				col: col,
			})
		}
	}
	return pixels
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
