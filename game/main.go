package game

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/ichibankunio/flib"
)

const (
	// Portrait 16:9 (HD)
	ScreenWidth  = 1080
	ScreenHeight = 1920
	SampleRate   = 44100
)

const (
	SceneMain flib.SceneID = iota
)

type Game struct {
	FlibGame *flib.Game
}

func (g *Game) Init() {
	g.FlibGame.AddScene(&MainScene{})
}

var once sync.Once

func (g *Game) Update() error {
	once.Do(g.Init)
	return g.FlibGame.Update()
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.FlibGame.Draw(screen)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS: %.2f", ebiten.CurrentFPS()), 12, 12)
}

func (g *Game) Layout(_, _ int) (int, int) {
	return ScreenWidth, ScreenHeight
}
