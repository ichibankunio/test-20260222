package game

func newStage7Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, playerX, playerY float64) {
		if frame%16 == 0 {
			offset := float64((frame / 16 * 7) % 360)
			d.spawnRing(ScreenWidth/2, 20, 1.0, 10, offset)
		}
		if frame%54 == 18 {
			d.spawnAimedSpread(ScreenWidth/2, 28, 1.08, []float64{-14, 0, 14}, playerX, playerY)
		}
		if frame%58 == 22 {
			xs := []float64{24, 120, 36, 108, 48, 96, 60, 84}
			d.spawnCoin(xs[(frame/58)%len(xs)], -6, 0, 1.0, 20)
		}
	})
}
