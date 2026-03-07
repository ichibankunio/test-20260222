package game

import (
	"fmt"
	"image/color"
	"sync"

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
	remaining    int
	pattern      int
	invincible   bool
	gameOver     bool
	gameClear    bool
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
	if s.handleDebugToggleInput() {
		return nil
	}

	if s.gameOver || s.gameClear {
		if isRestartInputJustPressed() {
			s.reset()
		}
		return nil
	}

	s.updatePlayerFromSwipe()
	if s.danmaku != nil {
		tick := s.danmaku.Update(s.playerX, s.playerY, s.playerRadius, s.invincible)
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
	}
	if s.remaining > 0 {
		s.remaining--
		if s.remaining == 0 {
			s.gameClear = true
			return nil
		}
	}
	s.updateDanmakuPattern()
	return nil
}

func (s *MainScene) Draw(screen *ebiten.Image) {
	screen.Fill(bgNight)
	drawBackdrop(screen)
	s.portrait.draw(screen)
	drawTimer(screen, s.remaining)
	if !s.gameOver {
		drawPlayer(screen, s.playerX, s.playerY, s.playerRadius)
	}
	if s.danmaku != nil {
		s.danmaku.Draw(screen)
	}
	s.particles.draw(screen)

	drawUITextAt(screen, "5 MIN SURVIVAL", 4, 4)
	drawUITextAt(screen, fmt.Sprintf("ART %02d%%", int(s.portrait.completionRate()*100)), 82, 4)
	drawDebugButton(screen, s.invincible)
	drawUITextAt(screen, "SWIPE/DRAG: MOVE", 4, 16)
	drawUITextAt(screen, "ESC: EXIT", 4, 28)
	if s.gameOver {
		drawUITextCentered(screen, "GAME OVER", 0, 112, ScreenWidth, 12, 16)
		drawUITextCentered(screen, "TAP/SPACE: RETRY", 0, 128, ScreenWidth, 12, 12)
	}
	if s.gameClear {
		drawUITextCentered(screen, "CLEAR!", 0, 104, ScreenWidth, 12, 16)
		drawUITextCentered(screen, "お宝画像ゲット!", 0, 120, ScreenWidth, 12, 12)
		drawUITextCentered(screen, "TAP/SPACE: PLAY AGAIN", 0, 136, ScreenWidth, 12, 12)
	}
}

var (
	bgNight       = color.RGBA{R: 7, G: 8, B: 12, A: 255}
	gridDark      = color.RGBA{R: 20, G: 24, B: 32, A: 255}
	uiBorder      = color.RGBA{R: 240, G: 240, B: 240, A: 255}
	uiGaugeBase   = color.RGBA{R: 5, G: 5, B: 8, A: 255}
	uiTimerPanel  = color.RGBA{R: 16, G: 22, B: 34, A: 220}
	uiTimerBorder = color.RGBA{R: 144, G: 196, B: 255, A: 255}
	uiTimerTrack  = color.RGBA{R: 40, G: 52, B: 74, A: 255}
	uiTimerFill   = color.RGBA{R: 84, G: 212, B: 248, A: 255}
	playerMain    = color.RGBA{R: 248, G: 232, B: 56, A: 255}
	playerAccent  = color.RGBA{R: 176, G: 128, B: 24, A: 255}
	enemyBullet   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	enemyBulletIn = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	coinMain      = color.RGBA{R: 255, G: 214, B: 64, A: 255}
	coinAccent    = color.RGBA{R: 212, G: 164, B: 32, A: 255}
)

const (
	playerMoveMargin = 10.0
	gameSeconds      = 5 * 60
	tps              = 60
	gameFrames       = gameSeconds * tps
	patternFrames    = 30 * tps
	patternCount     = 10
	debugBtnW        = 52.0
	debugBtnH        = 14.0
	debugBtnX        = ScreenWidth - debugBtnW - 4.0
	debugBtnY        = 4.0
)

func (s *MainScene) reset() {
	s.portrait = newPortraitBuilder()
	s.playerX = ScreenWidth / 2
	s.playerY = ScreenHeight - 36
	s.playerRadius = 6
	s.gameOver = false
	s.gameClear = false
	s.remaining = gameFrames
	s.pattern = -1
	s.touchActive = false
	s.mouseActive = false
	s.particles.reset()
	s.updateDanmakuPattern()
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

func (s *MainScene) updateDanmakuPattern() {
	elapsed := gameFrames - s.remaining
	next := (elapsed / patternFrames) % patternCount
	if next == s.pattern {
		return
	}
	s.pattern = next
	s.danmaku = newDanmakuForStage(s.pattern + 1)
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
	screen.DrawImage(backdropImage(), nil)
}

var (
	backdropOnce sync.Once
	backdropImg  *ebiten.Image
)

func backdropImage() *ebiten.Image {
	backdropOnce.Do(func() {
		img := ebiten.NewImage(ScreenWidth, ScreenHeight)
		for y := 0.0; y < ScreenHeight; y += 16 {
			for x := 0.0; x < ScreenWidth; x += 16 {
				shade := gridDark
				if int((x+y)/16)%2 == 0 {
					shade = color.RGBA{R: 14, G: 18, B: 26, A: 255}
				}
				drawFilledRect(img, x, y, 16, 16, shade)
			}
		}
		backdropImg = img
	})
	return backdropImg
}

func drawTimer(screen *ebiten.Image, remaining int) {
	leftX := 8.0
	topY := 36.0
	boxW := ScreenWidth - 16.0
	boxH := 24.0

	drawFilledRect(screen, leftX, topY, boxW, boxH, uiTimerPanel)
	var frame vector.Path
	frame.MoveTo(float32(leftX), float32(topY))
	frame.LineTo(float32(leftX+boxW), float32(topY))
	frame.LineTo(float32(leftX+boxW), float32(topY+boxH))
	frame.LineTo(float32(leftX), float32(topY+boxH))
	frame.Close()
	frameDrawOp := &vector.DrawPathOptions{}
	frameDrawOp.AntiAlias = true
	frameDrawOp.ColorScale.ScaleWithColor(uiTimerBorder)
	frameStroke := &vector.StrokeOptions{}
	frameStroke.Width = 1.5
	frameStroke.LineJoin = vector.LineJoinRound
	vector.StrokePath(screen, &frame, frameStroke, frameDrawOp)

	progress := clamp(float64(max(0, remaining))/float64(gameFrames), 0, 1)
	trackX := leftX + 4.0
	trackY := topY + 14.0
	trackW := boxW - 8.0
	trackH := 6.0
	drawFilledRect(screen, trackX, trackY, trackW, trackH, uiTimerTrack)
	fillW := trackW * progress
	if fillW > 0 {
		drawFilledRect(screen, trackX, trackY, fillW, trackH, uiTimerFill)
		vector.DrawFilledCircle(screen, float32(trackX+fillW), float32(trackY+trackH/2), float32(trackH/2), uiTimerFill, true)
	}

	seconds := max(0, remaining/tps)
	minute := seconds / 60
	second := seconds % 60
	drawUITextAt(screen, "TIME LEFT", int(leftX)+4, int(topY)+2)
	drawUITextCentered(screen, fmt.Sprintf("%02d:%02d", minute, second), int(leftX), int(topY)-2, int(boxW), int(boxH), 12)
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
	text.Draw(screen, body, GetTextFace(), op)
}

func drawUITextCentered(screen *ebiten.Image, body string, x, y, w, h, size int) {
	op := &text.DrawOptions{}
	if size > 0 && size != 12 {
		scale := float64(size) / 12.0
		op.GeoM.Scale(scale, scale)
	}
	op.GeoM.Translate(float64(x+w/2), float64(y+h/2))
	op.LayoutOptions.PrimaryAlign = text.AlignCenter
	op.LayoutOptions.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(uiBorder)
	text.Draw(screen, body, GetTextFace(), op)
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
	return (stage - 1) % patternCount
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
