package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ichibankunio/flib"
)

type MainScene struct {
	font *text.GoTextFace
}

func (s *MainScene) Init(_ *flib.Game) {
	s.font = GetGoTextFace(28)
	if bgm := FirstBGM(); bgm != nil {
		bgm.SetVolume(0.3)
		bgm.Play()
	}
}

func (s *MainScene) Start(_ *flib.Game) {}

func (s *MainScene) Update(_ *flib.Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if se := FirstSE(); len(se) > 0 {
			PlaySE(se)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit")
	}
	return nil
}

func (s *MainScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	if img := GetImage("zentablue.png"); img != nil {
		op := &ebiten.DrawImageOptions{}
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		op.GeoM.Scale(2.0, 2.0)
		op.GeoM.Translate(float64(ScreenWidth/2-w), 120)
		screen.DrawImage(img, op)
		_ = h
	}

	ebitenutil.DebugPrintAt(screen, "Hello, mobile-game-template", 40, 420)
	ebitenutil.DebugPrintAt(screen, "hello codex!!!", 40, 440)
	ebitenutil.DebugPrintAt(screen, "Assets ready: image / font / json / audio", 40, 460)
	ebitenutil.DebugPrintAt(screen, "Press SPACE for SE, ESC to exit", 40, 500)

	if sample, ok := GetJSON("sample.json"); ok {
		ebitenutil.DebugPrintAt(screen, "JSON: "+sample.Title+" v"+sample.Version, 40, 540)
	}

	if s.font != nil {
		op := &text.DrawOptions{}
		op.GeoM.Translate(40, 610)
		text.Draw(screen, "Hello codex\nhello codex!!!", s.font, op)
	}
}

func (s *MainScene) GetStatus() int { return 0 }

func (s *MainScene) GetID() flib.SceneID { return SceneMain }
