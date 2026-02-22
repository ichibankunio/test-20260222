package game

func newStage4Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, _, _ float64) {
		if frame%18 == 0 {
			offset := float64((frame / 18 * 9) % 360)
			d.spawnRing(ScreenWidth/2, 36, 0.95, 8, offset)
		}
		if frame%21 == 1 {
			d.spawnCoin(ScreenWidth/2, -6, 0, 0.9, 24)
		}
	})
}
