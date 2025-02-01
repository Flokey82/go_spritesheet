package gospritesheet

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"math/rand"
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

// interpolateColor returns a color interpolated between colorA and colorB at a given percentage (0.0 - 1.0).
func interpolateColor(colorA, colorB color.Color, percentage float64) color.Color {
	rA, gA, bA, aA := colorA.RGBA()
	rB, gB, bB, aB := colorB.RGBA()

	r := uint16(float64(rA) + percentage*(float64(rB)-float64(rA)))
	g := uint16(float64(gA) + percentage*(float64(gB)-float64(gA)))
	b := uint16(float64(bA) + percentage*(float64(bB)-float64(bA)))
	a := uint16(float64(aA) + percentage*(float64(aB)-float64(aA)))

	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// ApplyFlameEffect generates a flame effect for the given layer.
func ApplyFlameEffect(layer image.Image, colorA, colorB color.Color) *image.RGBA {
	bounds := layer.Bounds()
	flameLayer := image.NewRGBA(bounds)

	// Build a gradient index for all the colors we want to use.
	const numColors = 10
	gradient := make([]color.Color, numColors)
	for i := 0; i < numColors; i++ {
		percentage := float64(i) / float64(numColors-1)
		gradient[i] = interpolateColor(colorA, colorB, percentage)
	}
	// We iterate over the pixels bottom to top and look at the pixel below.
	// If the pixel below is set on the original layer, we set the current pixel on the flame layer
	// to the first color in the gradient.
	// Else if the pixel below is set on the flame layer, we set the current pixel to the next color in the gradient.
	for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get the color of the current pixel on the original layer
			_, _, _, a := layer.At(x, y).RGBA()
			_, _, _, aBelow := layer.At(x, y+1).RGBA()
			_, _, _, aFlameBelow := flameLayer.At(x, y+1).RGBA()

			// If the pixel is set on the original layer, we skip it since we don't want to cover it.
			if a != 0 {
				continue
			}

			// If the pixel below is set on the original layer, set the current pixel on the flame layer to the first color in the gradient
			if aBelow != 0 {
				flameLayer.Set(x, y, gradient[0])
			} else if aFlameBelow != 0 {
				// Check if at least two neighboring pixels are set on either layer.
				// If not, skip the current pixel.
				var numNeighbors int
				for i := -1; i <= 1; i++ {
					if x+i < bounds.Min.X || x+i >= bounds.Max.X {
						continue
					}
					for j := -1; j <= 1; j++ {
						if y+j < bounds.Min.Y || y+j >= bounds.Max.Y {
							continue
						}

						_, _, _, aLeft := layer.At(x+i, y+j).RGBA()
						_, _, _, aFlameLeft := flameLayer.At(x+i, y+j).RGBA()
						if aLeft != 0 || aFlameLeft != 0 {
							numNeighbors++
						}
					}
				}

				if (numNeighbors < 3 && rand.Float64() < 0.5) || rand.Float64() < 0.1 {
					continue
				}

				// Get the index of the current color in the gradient.
				index := 0
				for i, c := range gradient {
					if c == flameLayer.At(x, y+1) {
						index = i
						break
					}
				}

				// Set the current pixel to the next color in the gradient
				if index < numColors-1 {
					flameLayer.Set(x, y, gradient[index+1])
				}
			}
		}
	}

	return flameLayer
}

// blendColors blends two colors with a given intensity (0.0 - 1.0).
func blendColors(colorA, colorB color.Color, intensity float64) color.Color {
	rA, gA, bA, aA := colorA.RGBA()
	rB, gB, bB, aB := colorB.RGBA()

	r := uint16(float64(rA)*(1.0-intensity) + float64(rB)*intensity)
	g := uint16(float64(gA)*(1.0-intensity) + float64(gB)*intensity)
	b := uint16(float64(bA)*(1.0-intensity) + float64(bB)*intensity)
	a := uint16(float64(aA)*(1.0-intensity) + float64(aB)*intensity)

	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}
