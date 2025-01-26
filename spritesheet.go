package gospritesheet

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
)

// ReplaceColor replaces all pixels of a certain color in an image with another color.
func ReplaceColor(img image.Image, from, to color.Color) *image.RGBA {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)
	draw.Draw(newImg, bounds, img, bounds.Min, draw.Src)

	for x := 0; x < bounds.Dx(); x++ {
		for y := 0; y < bounds.Dy(); y++ {
			if img.At(x, y) == from {
				newImg.Set(x, y, to)
			}
		}
	}

	return newImg
}

// Spritesheet is a convenience wrapper around locating sprites in a spritesheet.
type Spritesheet struct {
	image    image.Image
	tileSize int // Size of each tile in the spritesheet
	xCount   int // Number of tiles in the x direction
	yCount   int // Number of tiles in the y direction
	x        int // Width of the spritesheet
	y        int // Height of the spritesheet
}

func New(imgData []byte, tileSize int) (*Spritesheet, error) {
	// Decode an image from the image file's byte slice.
	// Now the byte slice is generated with //go:generate for Go 1.15 or older.
	// If you use Go 1.16 or newer, it is strongly recommended to use //go:embed to embed the image file.
	// See https://pkg.go.dev/embed for more details.
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}

	// Get the size of the image
	bounds := img.Bounds()
	x := bounds.Dx()
	y := bounds.Dy()

	// Calculate the number of tiles in the x and y directions
	xCount := x / tileSize
	yCount := y / tileSize

	return &Spritesheet{
		image:    img,
		tileSize: tileSize,
		xCount:   xCount,
		yCount:   yCount,
		x:        x,
		y:        y,
	}, nil
}

// numTiles returns the number of tiles in the spritesheet.
func (s *Spritesheet) NumTiles() int {
	return s.xCount * s.yCount
}

// TileImage returns an image.Image of the tile at the given index.
// TODO: This should maybe take an image (and maybe offset) to draw on
// instead of returning a new image. Also the color replacement could be
// done here.
func (s *Spritesheet) TileImage(index int) image.Image {
	// Calculate the x and y position of the tile in the spritesheet
	x := (index % s.xCount) * s.tileSize
	y := (index / s.xCount) * s.tileSize

	// Create a new RGBA image for the tile
	tile := image.NewRGBA(image.Rect(0, 0, s.tileSize, s.tileSize))

	// Copy the tile from the spritesheet to the new image
	for i := 0; i < s.tileSize; i++ {
		for j := 0; j < s.tileSize; j++ {
			tile.Set(i, j, s.image.At(x+i, y+j))
		}
	}

	return tile
}
