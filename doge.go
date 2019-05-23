// Copyright 2018 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build example jsgo

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/vorbis"
	"github.com/hajimehoshi/ebiten/audio/wav"
	raudio "github.com/hajimehoshi/ebiten/examples/resources/audio"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	resources "github.com/hajimehoshi/ebiten/examples/resources/images/doge"
	"github.com/hajimehoshi/ebiten/inpututil"
	"github.com/hajimehoshi/ebiten/text"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func floorDiv(x, y int) int {
	d := x / y
	if d*y == x || x >= 0 {
		return d
	}
	return d - 1
}

func floorMod(x, y int) int {
	return x - floorDiv(x, y)*y
}

const (
	screenWidth      = 640
	screenHeight     = 480
	tileSize         = 32
	fontSize         = 32
  rockHeight       = 55
  cactusHeight       = 65
  spikeHeight       = 30
	smallFontSize    = fontSize / 2
	obstacleWidth        = 58
	obstacleStartOffsetX = 8
  obstacleIntervalX = 8
)

var (
	dogeImage     *ebiten.Image
	tilesImage      *ebiten.Image
  rockImage      *ebiten.Image
  cactusImage      *ebiten.Image
  fireballImage      *ebiten.Image
  backgroundImage      *ebiten.Image
  spikeImage      *ebiten.Image
	arcadeFont      font.Face
	smallArcadeFont font.Face
)

type obstacle struct {
    height float64
    image *ebiten.Image
}

func init() {
	img, _, err := image.Decode(bytes.NewReader(resources.Doge_png))
	if err != nil {
		log.Fatal(err)
	}
	dogeImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

  img, _, err = image.Decode(bytes.NewReader(resources.Background_png))
	if err != nil {
		log.Fatal(err)
	}
	backgroundImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)


	img, _, err = image.Decode(bytes.NewReader(resources.Tiles_png))
	if err != nil {
		log.Fatal(err)
	}
	tilesImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

  img, _, err = image.Decode(bytes.NewReader(resources.Rock_png))
	if err != nil {
		log.Fatal(err)
	}
	rockImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

  img, _, err = image.Decode(bytes.NewReader(resources.Cactus_png))
	if err != nil {
		log.Fatal(err)
	}
	cactusImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

  img, _, err = image.Decode(bytes.NewReader(resources.Spike_png))
	if err != nil {
		log.Fatal(err)
	}
	spikeImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)

  img, _, err = image.Decode(bytes.NewReader(resources.Fireball_png))
	if err != nil {
		log.Fatal(err)
	}
	fireballImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
}

func init() {
	tt, err := truetype.Parse(fonts.ArcadeN_ttf)
	if err != nil {
		log.Fatal(err)
	}
	const dpi = 72
	arcadeFont = truetype.NewFace(tt, &truetype.Options{
		Size:    fontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	smallArcadeFont = truetype.NewFace(tt, &truetype.Options{
		Size:    smallFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}

var (
	audioContext *audio.Context
	jumpPlayer   *audio.Player
	hitPlayer    *audio.Player
)

func init() {
	audioContext, _ = audio.NewContext(44100)

	jumpD, err := vorbis.Decode(audioContext, audio.BytesReadSeekCloser(raudio.Jump_ogg))
	if err != nil {
		log.Fatal(err)
	}
	jumpPlayer, err = audio.NewPlayer(audioContext, jumpD)
	if err != nil {
		log.Fatal(err)
	}

	jabD, err := wav.Decode(audioContext, audio.BytesReadSeekCloser(raudio.Jab_wav))
	if err != nil {
		log.Fatal(err)
	}
	hitPlayer, err = audio.NewPlayer(audioContext, jabD)
	if err != nil {
		log.Fatal(err)
	}
}

type Mode int

const (
	ModeTitle Mode = iota
	ModeGame
	ModeGameOver
)

type Game struct {
	mode Mode

	// The doge's position
	x16  int
	y16  int
	vy16 int

	// Camera
	cameraX int
	cameraY int

  obstacleXs []int

  // Jumps left
  jumpsLeft int

  obstacle_cache map[int]obstacle

  //Fireball position
  fx16 int
  fy16 int



  //Gameover count
	gameoverCount int
}

func NewGame() *Game {
	g := &Game{}
	g.init()
	return g
}

func (g *Game) init() {
	g.x16 = 0
	g.y16 = 5100
  g.fx16 = 10000
  g.fy16 = 4700
	g.cameraX = -240
	g.cameraY = 0
  g.jumpsLeft = 4
  g.obstacle_cache = make(map[int]obstacle)
  g.obstacleXs = make([]int, 256)
  for i := range g.obstacleXs {
    g.obstacleXs[i] = rand.Intn(3)
  }
  obs1 := new(obstacle)
  obs1.height = rockHeight
  obs1.image = rockImage
  obs2 := new(obstacle)
  obs2.height = cactusHeight
  obs2.image = cactusImage
  obs3 := new(obstacle)
  obs3.height = spikeHeight
  obs3.image = spikeImage
  g.obstacle_cache[0] = *obs1
  g.obstacle_cache[1] = *obs2
  g.obstacle_cache[2] = *obs3
}

func jump() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	if len(inpututil.JustPressedTouchIDs()) > 0 {
		return true
	}
	return false
}

func (g *Game) Update(screen *ebiten.Image) error {
	switch g.mode {
	case ModeTitle:
		if jump() {
			g.mode = ModeGame
		}
	case ModeGame:
		g.x16 += 64
    g.fx16 -= 32
    if g.x16 - g.fx16 > 3000 {
      g.fx16 = g.x16 + 10000
      g.fy16 = 4500 - rand.Intn(600)
    }
		g.cameraX += 4
		if jump() && g.jumpsLeft > 0{
      g.jumpsLeft -= 1
			g.vy16 = -100
			jumpPlayer.Rewind()
			jumpPlayer.Play()
		}
		g.y16 += g.vy16
    if g.y16 > 5600 {
      g.y16 = 5600
      if g.jumpsLeft == 0{
        g.jumpsLeft = 4
      }
    }

		// Gravity
		g.vy16 += 4
		if g.vy16 > 96 {
			g.vy16 = 96
		}

		if g.hit() {
			hitPlayer.Rewind()
			hitPlayer.Play()
			g.mode = ModeGameOver
			g.gameoverCount = 30
		}
	case ModeGameOver:
		if g.gameoverCount > 0 {
			g.gameoverCount--
		}
		if g.gameoverCount == 0 && jump() {
			g.init()
			g.mode = ModeTitle
		}
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	// screen.Fill(color.RGBA{0x4E, 0xAB, 0xFD, 0xff})
  // Draws Background Image
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.5, 0.5)
	screen.DrawImage(backgroundImage, op)
	g.drawTiles(screen)

	if g.mode != ModeTitle {
		g.drawDoge(screen)
    g.drawFireball(screen)
	}
	var texts []string
	switch g.mode {
	case ModeTitle:
		texts = []string{"DOGE DODGE", "", "", "", "", "PRESS SPACE KEY", "", "OR TOUCH SCREEN"}
	case ModeGameOver:
		texts = []string{"", "GAME OVER!"}
	}
	for i, l := range texts {
		x := (screenWidth - len(l)*fontSize) / 2
		text.Draw(screen, l, arcadeFont, x, (i+4)*fontSize, color.White)
	}

	if g.mode == ModeTitle {
		msg := []string{
			"4 JUMPS TILL YOU HAVE TO LAND",
		}
		for i, l := range msg {
			x := (screenWidth - len(l)*smallFontSize) / 2
			text.Draw(screen, l, smallArcadeFont, x, screenHeight-4+(i-1)*smallFontSize, color.White)
		}
	}

	scoreStr := fmt.Sprintf("%04d", g.score())
	text.Draw(screen, scoreStr, arcadeFont, screenWidth-len(scoreStr)*fontSize, fontSize, color.White)
	return nil
}

func (g *Game) obstacleAt(tileX int) (obs obstacle, ok bool) {
	if (tileX - obstacleStartOffsetX) <= 0 {
		return g.obstacle_cache[0], false
	}
	if floorMod(tileX-obstacleStartOffsetX, obstacleIntervalX) != 0 {
		return g.obstacle_cache[0], false
	}
  idx := (floorDiv(tileX-obstacleStartOffsetX, obstacleIntervalX))%len(g.obstacleXs)
  return g.obstacle_cache[g.obstacleXs[idx]], true
}

func (g *Game) score() int {
	x := floorDiv(g.x16, 16) / tileSize
	if (x - obstacleStartOffsetX) <= 0 {
		return 0
	}
	return floorDiv(x-obstacleStartOffsetX, obstacleIntervalX)
}

func (g *Game) hit() bool {
  if g.mode != ModeGame {
		return false
	}
  const (
		dogeWidth  = 30
		dogeHeight = 60
	)
	w, h := dogeImage.Size()
	x0 := floorDiv(g.x16, 16) + (w-dogeWidth)/2
	y0 := floorDiv(g.y16, 16) + (h-dogeHeight)/2
	x1 := x0 + dogeWidth
	y1 := y0 + dogeHeight
	if y0 < -tileSize*4 {
		return true
	}
	if y1 >= screenHeight-tileSize {
		return true
	}
  if x0 <= floorDiv(g.fx16, 16) &&  floorDiv(g.fx16, 16) <= x1 &&  y0 <= floorDiv(g.fy16, 16) &&  floorDiv(g.fy16, 16) <= y1 {
    return true
  }
	xMin := floorDiv(x0-obstacleWidth, tileSize)
	xMax := floorDiv(x0+dogeWidth, tileSize)
	for x := xMin; x <= xMax; x++ {
		obs, ok := g.obstacleAt(x)
		if !ok {
			continue
		}
		if x0 >= x*tileSize+obstacleWidth {
			continue
		}
		if x1 < x*tileSize {
			continue
		}
		if y0 > 372 - int(obs.height){
			return true
		}
	}
	return false
}

func (g *Game) drawTiles(screen *ebiten.Image) {
	const (
		nx           = screenWidth / tileSize
		ny           = screenHeight / tileSize
	)

	op := &ebiten.DrawImageOptions{}
	for i := -2; i < nx+1; i++ {
		// ground
		op.GeoM.Reset()
		op.GeoM.Translate(float64(i*tileSize-floorMod(g.cameraX, tileSize)),
			float64((ny-1)*tileSize-floorMod(g.cameraY, tileSize)))
		screen.DrawImage(tilesImage.SubImage(image.Rect(0, 0, tileSize, tileSize)).(*ebiten.Image), op)
    if obj, ok := g.obstacleAt(floorDiv(g.cameraX, tileSize) + i); ok {
      op.GeoM.Translate(0,-obj.height)
      screen.DrawImage(obj.image, op)

    }
	}


}

func (g *Game) drawFireball(screen *ebiten.Image) {
  op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(g.fx16/16.0)-float64(g.cameraX), float64(g.fy16/16.0)-float64(g.cameraY))
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(fireballImage, op)

}

func (g *Game) drawDoge(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(g.x16/16.0)-float64(g.cameraX), float64(g.y16/16.0)-float64(g.cameraY))
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(dogeImage, op)
}

func main() {
	g := NewGame()
	// On browsers, let's use fullscreen so that this is playable on any browsers.
	// It is planned to ignore the given 'scale' apply fullscreen automatically on browsers (#571).
	if runtime.GOARCH == "js" || runtime.GOOS == "js" {
		ebiten.SetFullscreen(true)
	}
	if err := ebiten.Run(g.Update, screenWidth, screenHeight, 1, "Doge Dodge"); err != nil {
		panic(err)
	}
}
