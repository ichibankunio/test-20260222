package game

func newDanmakuForStage(stage int) Danmaku {
	switch stageNumber(stage) {
	case 0:
		return newStage1Danmaku()
	case 1:
		return newStage2Danmaku()
	case 2:
		return newStage3Danmaku()
	case 3:
		return newStage4Danmaku()
	case 4:
		return newStage5Danmaku()
	case 5:
		return newStage6Danmaku()
	case 6:
		return newStage7Danmaku()
	case 7:
		return newStage8Danmaku()
	case 8:
		return newStage9Danmaku()
	default:
		return newStage10Danmaku()
	}
}
