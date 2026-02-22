package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/ichibankunio/flib"
)

type MainScene struct {
	playerX        float64
	playerY        float64
	playerVY       float64
	cameraX        float64
	obstacles      []obstacle
	nextObstacleX  float64
	obstacleSerial int
	gameOver       bool
}

type obstacle struct {
	x      float64
	width  float64
	height float64
}

func (s *MainScene) Init(_ *flib.Game) {
	s.reset()
	if bgm := FirstBGM(); bgm != nil {
		bgm.SetVolume(0.3)
		bgm.Play()
	}
}

func (s *MainScene) Start(_ *flib.Game) {}

func (s *MainScene) Update(_ *flib.Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit")
	}

	if s.gameOver {
		if isRestartInputJustPressed() {
			s.reset()
		}
		return nil
	}

	s.cameraX += autoScrollSpeed
	s.playerX = s.cameraX + playerScreenOffsetX

	onGround := s.playerY >= groundTop()-playerHeight
	if onGround && isJumpInputJustPressed() {
		s.playerVY = jumpVelocity
		if se := FirstSE(); len(se) > 0 {
			PlaySE(se)
		}
	}
	s.playerVY += gravity
	s.playerY += s.playerVY

	floorY := groundTop() - playerHeight
	if s.playerY > floorY {
		s.playerY = floorY
		s.playerVY = 0
	}

	s.spawnObstacles()
	s.cleanupObstacles()
	if s.hitObstacle() {
		s.gameOver = true
	}

	return nil
}

func (s *MainScene) Draw(screen *ebiten.Image) {
	screen.Fill(gbLightest)

	drawClouds(screen, s.cameraX)
	drawGround(screen, s.cameraX)
	drawObstacles(screen, s.cameraX, s.obstacles)
	drawPlayer(screen, s.playerX-s.cameraX, s.playerY)

	ebitenutil.DebugPrintAt(screen, "JUMP: TAP/SPACE", 4, 4)
	ebitenutil.DebugPrintAt(screen, "ESC: EXIT", 4, 16)
	if s.gameOver {
		ebitenutil.DebugPrintAt(screen, "GAME OVER", 30, 104)
		ebitenutil.DebugPrintAt(screen, "TAP/SPACE: RETRY", 16, 118)
	}
}

var (
	gbDarkest  = color.RGBA{R: 15, G: 56, B: 15, A: 255}
	gbDark     = color.RGBA{R: 48, G: 98, B: 48, A: 255}
	gbLight    = color.RGBA{R: 139, G: 172, B: 15, A: 255}
	gbLightest = color.RGBA{R: 155, G: 188, B: 15, A: 255}
)

const (
	groundH             = 66.0
	playerWidth         = 18.0
	playerHeight        = 24.0
	autoScrollSpeed     = 1.3
	gravity             = 0.44
	jumpVelocity        = -8.0
	groundTileW         = 24.0
	playerScreenOffsetX = 24.0
	obstacleMinGap      = 96.0
	obstacleWidth       = 14.0
)

func groundTop() float64 {
	return ScreenHeight - groundH
}

func drawClouds(screen *ebiten.Image, cameraX float64) {
	const cloudSpacing = 96.0
	cloudCameraX := cameraX * 0.35
	start := -math.Mod(cloudCameraX, cloudSpacing)
	for x := start - cloudSpacing; x < ScreenWidth+cloudSpacing; x += cloudSpacing {
		screenX := x + 12
		drawFilledRect(screen, screenX, 40, 42, 12, gbLight)
		drawFilledRect(screen, screenX+8, 32, 30, 9, gbLight)
	}
}

func drawGround(screen *ebiten.Image, cameraX float64) {
	drawFilledRect(screen, 0, groundTop()-3, ScreenWidth, 3, gbLight)
	start := -math.Mod(cameraX, groundTileW)
	for x := start - groundTileW; x < ScreenWidth+groundTileW; x += groundTileW {
		tile := int((cameraX + x) / groundTileW)
		fill := gbDark
		if tile%2 == 0 {
			fill = gbDarkest
		}
		drawFilledRect(screen, x, groundTop(), groundTileW, groundH, fill)
		drawFilledRect(screen, x+3, groundTop()+4, 5, 5, gbLightest)
	}
}

func drawPlayer(screen *ebiten.Image, x, y float64) {
	drawFilledRect(screen, x, y, playerWidth, playerHeight, gbDark)
	drawFilledRect(screen, x, y, playerWidth, 6, gbDarkest)
	drawFilledRect(screen, x+5, y+7, 8, 8, gbLightest)
	drawFilledRect(screen, x+2, y+16, 5, 8, gbLight)
	drawFilledRect(screen, x+11, y+16, 5, 8, gbLight)
}

func drawObstacles(screen *ebiten.Image, cameraX float64, obstacles []obstacle) {
	for _, o := range obstacles {
		x := o.x - cameraX
		y := groundTop() - o.height
		drawFilledRect(screen, x, y, o.width, o.height, gbDarkest)
		drawFilledRect(screen, x+2, y+3, o.width-4, 4, gbDark)
	}
}

func (s *MainScene) reset() {
	s.cameraX = 0
	s.playerX = playerScreenOffsetX
	s.playerY = groundTop() - playerHeight
	s.playerVY = 0
	s.obstacles = s.obstacles[:0]
	s.nextObstacleX = ScreenWidth + obstacleMinGap
	s.obstacleSerial = 0
	s.gameOver = false
	s.spawnObstacles()
}

func (s *MainScene) spawnObstacles() {
	for s.nextObstacleX < s.cameraX+ScreenWidth*2 {
		s.obstacles = append(s.obstacles, obstacle{
			x:      s.nextObstacleX,
			width:  obstacleWidth,
			height: obstacleHeightFromIndex(s.obstacleSerial),
		})
		s.obstacleSerial++
		s.nextObstacleX += obstacleGapFromIndex(s.obstacleSerial)
	}
}

func obstacleHeightFromIndex(i int) float64 {
	heights := []float64{20, 30, 42, 26}
	return heights[i%len(heights)]
}

func obstacleGapFromIndex(i int) float64 {
	gaps := []float64{88, 102, 120, 96}
	return gaps[i%len(gaps)]
}

func (s *MainScene) cleanupObstacles() {
	keepFrom := 0
	for keepFrom < len(s.obstacles) {
		if s.obstacles[keepFrom].x+s.obstacles[keepFrom].width >= s.cameraX-obstacleMinGap {
			break
		}
		keepFrom++
	}
	if keepFrom > 0 {
		copy(s.obstacles, s.obstacles[keepFrom:])
		s.obstacles = s.obstacles[:len(s.obstacles)-keepFrom]
	}
}

func (s *MainScene) hitObstacle() bool {
	playerLeft := s.playerX
	playerRight := s.playerX + playerWidth
	playerTop := s.playerY
	playerBottom := s.playerY + playerHeight

	for _, o := range s.obstacles {
		obsLeft := o.x
		obsRight := o.x + o.width
		obsTop := groundTop() - o.height
		obsBottom := groundTop()
		if playerRight <= obsLeft || playerLeft >= obsRight || playerBottom <= obsTop || playerTop >= obsBottom {
			continue
		}
		return true
	}
	return false
}

func drawFilledRect(screen *ebiten.Image, x, y, width, height float64, c color.Color) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(height), c, true)
}

func isJumpInputJustPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	return len(inpututil.AppendJustPressedTouchIDs(nil)) > 0
}

func isRestartInputJustPressed() bool {
	return isJumpInputJustPressed()
}

func (s *MainScene) GetStatus() int { return 0 }

func (s *MainScene) GetID() flib.SceneID { return SceneMain }
