package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ichibankunio/flib"
	"github.com/ichibankunio/test-20260222/game"
)

func main() {
	ebiten.SetWindowSize(432, 768)
	ebiten.SetWindowTitle("5分間弾幕避けたらお宝画像ゲット")

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
