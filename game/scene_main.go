package game

import (
	"fmt"
	"image/color"
	"time"

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
	invincible   bool
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
	perfProbe    framePerfProbe
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
	if s.perfProbe.enabled {
		start := time.Now()
		defer s.perfProbe.updateTotal.update(time.Since(start))
	}

	s.perfProbe.measure(&s.perfProbe.updateParticles, func() {
		s.particles.update()
	})
	s.perfProbe.measure(&s.perfProbe.updatePortrait, func() {
		s.portrait.update()
	})

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		s.perfProbe.enabled = !s.perfProbe.enabled
	}
	if s.handleDebugToggleInput() {
		return nil
	}

	if s.gameOver {
		if isRestartInputJustPressed() {
			s.retryStage()
		}
		return nil
	}

	s.updatePlayerFromSwipe()
	if s.danmaku != nil {
		var tick DanmakuTick
		s.perfProbe.measure(&s.perfProbe.updateDanmaku, func() {
			tick = s.danmaku.Update(s.playerX, s.playerY, s.playerRadius, s.invincible)
		})
		for _, p := range tick.CoinCollecteds {
			s.particles.spawnCoinPickup(p.X, p.Y)
			s.portrait.onCoinCollected(p.X, p.Y)
		}
		for _, p := range tick.BulletImpacts {
			s.particles.spawnBulletImpact(p.X, p.Y)
		}
		if tick.Hit {
			if s.invincible {
				return nil
			}
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
	s.perfProbe.measure(&s.perfProbe.drawTotal, func() {
		screen.Fill(bgNight)
		s.perfProbe.measure(&s.perfProbe.drawBackdrop, func() {
			drawBackdrop(screen)
		})
		s.perfProbe.measure(&s.perfProbe.drawPortrait, func() {
			s.portrait.draw(screen)
		})
		drawGauge(screen, s.gauge/stageGaugeMax, s.stage)
		if !s.gameOver {
			drawPlayer(screen, s.playerX, s.playerY, s.playerRadius)
		}
		if s.danmaku != nil {
			s.perfProbe.measure(&s.perfProbe.drawDanmaku, func() {
				s.danmaku.Draw(screen)
			})
		}
		s.perfProbe.measure(&s.perfProbe.drawParticles, func() {
			s.particles.draw(screen)
		})

		s.perfProbe.measure(&s.perfProbe.drawUI, func() {
			drawUITextAt(screen, fmt.Sprintf("STAGE %d", s.stage), 4, 4)
			drawUITextAt(screen, fmt.Sprintf("ART %02d%%", int(s.portrait.completionRate()*100)), 82, 4)
			drawDebugButton(screen, s.invincible)
			drawUITextAt(screen, "SWIPE/DRAG: MOVE", 4, 16)
			drawUITextAt(screen, "ESC: EXIT", 4, 28)
			drawUITextAt(screen, "F3: PERF HUD", 4, 40)
			if s.gameOver {
				drawUITextCentered(screen, "GAME OVER", 0, 112, ScreenWidth, 12, 16)
				drawUITextCentered(screen, "TAP/SPACE: RETRY", 0, 128, ScreenWidth, 12, 12)
			}
		})
	})
	s.drawPerfHUD(screen)
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
)

const (
	playerMoveMargin = 10.0
	stageGaugeMax    = 100.0
	stageCycleCount  = 10
	debugBtnW        = 52.0
	debugBtnH        = 14.0
	debugBtnX        = ScreenWidth - debugBtnW - 4.0
	debugBtnY        = 4.0
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

func (s *MainScene) handleDebugToggleInput() bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		gx, gy := s.toGamePosition(x, y)
		if isInDebugButton(gx, gy) {
			s.invincible = !s.invincible
			s.mouseActive = false
			s.touchActive = false
			return true
		}
	}
	touchIDs := inpututil.AppendJustPressedTouchIDs(nil)
	for _, id := range touchIDs {
		x, y := ebiten.TouchPosition(id)
		gx, gy := s.toGamePosition(x, y)
		if isInDebugButton(gx, gy) {
			s.invincible = !s.invincible
			s.mouseActive = false
			s.touchActive = false
			return true
		}
	}
	return false
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

func drawDebugButton(screen *ebiten.Image, invincible bool) {
	drawFilledRect(screen, debugBtnX, debugBtnY, debugBtnW, debugBtnH, uiGaugeBase)
	path := &vector.Path{}
	path.MoveTo(float32(debugBtnX), float32(debugBtnY))
	path.LineTo(float32(debugBtnX+debugBtnW), float32(debugBtnY))
	path.LineTo(float32(debugBtnX+debugBtnW), float32(debugBtnY+debugBtnH))
	path.LineTo(float32(debugBtnX), float32(debugBtnY+debugBtnH))
	path.Close()
	op := &vector.DrawPathOptions{}
	op.AntiAlias = true
	op.ColorScale.ScaleWithColor(uiBorder)
	stroke := &vector.StrokeOptions{}
	stroke.Width = 2
	vector.StrokePath(screen, path, stroke, op)

	label := "通常"
	if invincible {
		label = "無敵"
	}
	drawUITextCentered(screen, "DBG:"+label, int(debugBtnX), int(debugBtnY), int(debugBtnW), int(debugBtnH), 10)
}

func isInDebugButton(x, y float64) bool {
	return x >= debugBtnX && x <= debugBtnX+debugBtnW && y >= debugBtnY && y <= debugBtnY+debugBtnH
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
		vector.DrawFilledCircle(screen, float32(b.x), float32(b.y), float32(b.radius), enemyBullet, false)
		vector.DrawFilledCircle(screen, float32(b.x), float32(b.y), float32(b.radius*0.45), enemyBulletIn, false)
	}
}

func drawCoins(screen *ebiten.Image, coins []projectile) {
	if globalCoinShaderRenderer.draw(screen, coins) {
		return
	}
	for _, c := range coins {
		vector.DrawFilledCircle(screen, float32(c.x), float32(c.y), float32(c.radius), coinMain, true)
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

func (s *MainScene) drawPerfHUD(screen *ebiten.Image) {
	if !s.perfProbe.enabled {
		return
	}
	bullets, coins := danmakuProjectileCounts(s.danmaku)
	drawUITextAt(screen, fmt.Sprintf("PERF avg/max ms (target %.2f)", frameBudgetMs), 4, 184)
	drawUITextAt(screen, fmt.Sprintf("upd: %.2f/%.2f (dan %.2f)", s.perfProbe.updateTotal.avgMs, s.perfProbe.updateTotal.maxMs, s.perfProbe.updateDanmaku.avgMs), 4, 196)
	drawUITextAt(screen, fmt.Sprintf("drw: %.2f/%.2f (bg %.2f dan %.2f)", s.perfProbe.drawTotal.avgMs, s.perfProbe.drawTotal.maxMs, s.perfProbe.drawBackdrop.avgMs, s.perfProbe.drawDanmaku.avgMs), 4, 208)
	drawUITextAt(screen, fmt.Sprintf("prt: u%.2f d%.2f shd:%d", s.perfProbe.updatePortrait.avgMs, s.perfProbe.drawPortrait.avgMs, len(s.portrait.shards)), 4, 220)
	drawUITextAt(screen, fmt.Sprintf("ptc: u%.2f d%.2f n:%d", s.perfProbe.updateParticles.avgMs, s.perfProbe.drawParticles.avgMs, len(s.particles.items)), 4, 232)
	drawUITextAt(screen, fmt.Sprintf("blt:%d coin:%d", bullets, coins), 4, 244)
}
