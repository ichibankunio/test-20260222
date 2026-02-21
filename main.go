package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ichibankunio/flib"
	"github.com/ichibankunio/mobile-game-template/game"
)

func main() {
	ebiten.SetWindowSize(540, 960)
	ebiten.SetWindowTitle("mobile-game-template")

	mainGame := &game.Game{
		FlibGame: &flib.Game{
			State: 0,
			Lang:  flib.LANG_EN,
		},
	}
	mainGame.FlibGame.Storage.Init()

	if err := ebiten.RunGame(mainGame); err != nil {
		log.Fatal(err)
	}
}
