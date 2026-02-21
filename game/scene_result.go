package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/ichibankunio/flib"
)

type ResultScene struct {
	result RunResult
	tick   int
}

func (s *ResultScene) Init(_ *flib.Game) {}

func (s *ResultScene) Start(g *flib.Game) {
	s.tick = 0
	s.result = getLastRun(g)
}

func (s *ResultScene) Update(g *flib.Game) error {
	s.tick++
	if isJumpInputJustPressed() {
		flib.ShiftSceneWithFadeInOut(g, SceneMain, 28)
	}
	if ebiten.IsKeyPressed(ebiten.KeyT) {
		flib.ShiftSceneWithFadeInOut(g, SceneTitle, 28)
	}
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit")
	}
	return nil
}

func (s *ResultScene) Draw(screen *ebiten.Image) {
	drawSky(screen, float64(s.tick)*1.4, s.tick)
	drawHills(screen, float64(s.tick)*1.9)
	drawCitySilhouette(screen, float64(s.tick)*2.3)
	drawGround(screen, float64(s.tick)*2.6)
	drawOverlay(screen, color.RGBA{0, 0, 0, 110})

	drawFilledRect(screen, 110, 280, ScreenWidth-220, 760, color.RGBA{R: 9, G: 28, B: 54, A: 220})
	ebitenutil.DebugPrintAt(screen, "RESULT", 150, 340)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SCORE      %06d", s.result.Score), 150, 410)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("BEST       %06d", s.result.BestScore), 150, 446)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DISTANCE   %dm", s.result.Distance/12), 150, 482)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("MAX COMBO  x%d", s.result.MaxCombo), 150, 518)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SURVIVE    %.1fs", float64(s.result.SurviveTick)/60.0), 150, 554)

	ebitenutil.DebugPrintAt(screen, "SPACE / TAP: Retry", 150, 648)
	ebitenutil.DebugPrintAt(screen, "T: back to title", 150, 682)
	ebitenutil.DebugPrintAt(screen, "ESC: exit", 150, 716)
}

func (s *ResultScene) GetStatus() int { return 0 }

func (s *ResultScene) GetID() flib.SceneID { return SceneResult }
