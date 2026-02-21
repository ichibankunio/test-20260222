package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/ichibankunio/flib"
)

type TitleScene struct {
	tick      int
	bestScore int
}

func (s *TitleScene) Init(_ *flib.Game) {}

func (s *TitleScene) Start(g *flib.Game) {
	s.tick = 0
	s.bestScore = getBestScore(g)
}

func (s *TitleScene) Update(g *flib.Game) error {
	s.tick++
	EnsureBGMPlaying()
	if isJumpInputJustPressed() {
		flib.ShiftSceneWithFadeInOut(g, SceneMain, 28)
	}
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit")
	}
	return nil
}

func (s *TitleScene) Draw(screen *ebiten.Image) {
	drawSky(screen, float64(s.tick)*1.3, s.tick)
	drawHills(screen, float64(s.tick)*1.6)
	drawCitySilhouette(screen, float64(s.tick)*2.1)
	drawGround(screen, float64(s.tick)*2.5)

	drawFilledRect(screen, 84, 210, ScreenWidth-168, 640, color.RGBA{R: 8, G: 24, B: 44, A: 180})
	ebitenutil.DebugPrintAt(screen, "SKYLINE SPRINT", 130, 280)
	ebitenutil.DebugPrintAt(screen, "One-button action runner prototype", 130, 320)
	ebitenutil.DebugPrintAt(screen, "Avoid obstacles and collect sparks", 130, 354)
	ebitenutil.DebugPrintAt(screen, "to build combo and score.", 130, 378)
	ebitenutil.DebugPrintAt(screen, "", 130, 412)
	ebitenutil.DebugPrintAt(screen, "SPACE / TAP / CLICK: Jump", 130, 446)
	ebitenutil.DebugPrintAt(screen, "HOLD: longer airtime", 130, 470)
	ebitenutil.DebugPrintAt(screen, "R in game: quick retry", 130, 494)
	ebitenutil.DebugPrintAt(screen, "ESC: exit", 130, 518)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("BEST SCORE: %06d", s.bestScore), 130, 572)

	if math.Sin(float64(s.tick)*0.1) > -0.2 {
		ebitenutil.DebugPrintAt(screen, "TAP TO START", 130, 700)
	}
}

func (s *TitleScene) GetStatus() int { return 0 }

func (s *TitleScene) GetID() flib.SceneID { return SceneTitle }
