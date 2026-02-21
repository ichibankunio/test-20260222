package game

import "github.com/ichibankunio/flib"

type RunResult struct {
	Score       int
	BestScore   int
	Distance    int
	MaxCombo    int
	SurviveTick int
}

func setLastRun(g *flib.Game, result RunResult) {
	if g == nil {
		return
	}
	g.Storage.SetItem("last_run", result)
}

func getLastRun(g *flib.Game) RunResult {
	if g == nil {
		return RunResult{}
	}
	v := g.Storage.GetItem("last_run")
	if r, ok := v.(RunResult); ok {
		return r
	}
	return RunResult{}
}

func updateBestScore(g *flib.Game, score int) int {
	if g == nil {
		return score
	}
	best := getBestScore(g)
	if score > best {
		best = score
		g.Storage.SetItem("best_score", best)
	}
	return best
}

func getBestScore(g *flib.Game) int {
	if g == nil {
		return 0
	}
	v := g.Storage.GetItem("best_score")
	if v == nil {
		return 0
	}
	best, ok := v.(int)
	if !ok {
		return 0
	}
	return best
}
