package mobile

import (
	"github.com/hajimehoshi/ebiten/v2/mobile"
	"github.com/ichibankunio/flib"
	"github.com/ichibankunio/test-20260222/game"
)

var mainGame *game.Game

func init() {
	mainGame = &game.Game{
		FlibGame: &flib.Game{State: 0, Lang: flib.LANG_EN},
	}
	mainGame.FlibGame.Storage.Init()
	mobile.SetGame(mainGame)
}

// Dummy forces gomobile to compile this package.
func Dummy() {}
