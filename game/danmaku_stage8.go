package game

func newStage8Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, playerX, playerY float64) {
		if frame%24 == 0 {
			d.spawnAimedSpread(12, 30, 1.02, []float64{-12, 12}, playerX, playerY)
			d.spawnAimedSpread(ScreenWidth-12, 30, 1.02, []float64{-12, 12}, playerX, playerY)
		}
		if frame%44 == 14 {
			offset := float64((frame / 44 * 15) % 360)
			d.spawnRing(ScreenWidth/2, 54, 0.92, 7, offset)
		}
		if frame%52 == 18 {
			xs := []float64{18, 36, 54, 72, 90, 108, 126}
			d.spawnCoin(xs[(frame/52)%len(xs)], -6, 0, 1.04, 20)
		}
	})
}
