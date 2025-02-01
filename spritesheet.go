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

const (
	DirectionUp   = 1
	DirectionDown = -1
)

// applyEffect applies a generic effect (flame or drip) to the given layer.
func applyEffect(layer image.Image, colorA, colorB color.Color, numColors int, direction int) *image.RGBA {
	const minNeighbors = 3

	bounds := layer.Bounds()
	effectLayer := image.NewRGBA(bounds)

	// Build a gradient index for all the colors we want to use.
	gradient := make([]color.Color, numColors)
	for i := 0; i < numColors; i++ {
		percentage := float64(i) / float64(numColors-1)
		gradient[i] = interpolateColor(colorA, colorB, percentage)
	}

	// Iterate over the pixels in the specified direction
	startY := bounds.Max.Y - 1
	if direction == -1 {
		startY = bounds.Min.Y
	}
	for y := startY; y >= bounds.Min.Y && y < bounds.Max.Y; y -= direction {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get the color of the current pixel on the original layer
			_, _, _, a := layer.At(x, y).RGBA()
			_, _, _, aNeighbor := layer.At(x, y+direction).RGBA()
			_, _, _, aEffectNeighbor := effectLayer.At(x, y+direction).RGBA()

			// If the pixel is set on the original layer, we skip it since we don't want to cover it.
			if a != 0 {
				continue
			}

			// If the pixel above/below is set on the original layer, set the current pixel on the effect layer to the first color in the gradient
			if aNeighbor != 0 {
				effectLayer.Set(x, y, gradient[0])
			} else if aEffectNeighbor != 0 {
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
						_, _, _, aEffectLeft := effectLayer.At(x+i, y+j).RGBA()
						if aLeft != 0 || aEffectLeft != 0 {
							numNeighbors++
						}
					}
				}

				if (numNeighbors < minNeighbors && rand.Float64() < 0.5) || rand.Float64() < 0.2 {
					continue
				}

				// Get the index of the current color in the gradient.
				var index int
				for i, c := range gradient {
					if c == effectLayer.At(x, y+direction) {
						index = i
						break
					}
				}

				// Set the current pixel to the next color in the gradient
				if index < numColors-1 {
					effectLayer.Set(x, y, gradient[index+1])
				}
			}
		}
	}

	return effectLayer
}

// ApplyGlowEffect generates a glow effect for the given layer.
func ApplyGlowEffect(layer image.Image, colorA, colorB color.Color) *image.RGBA {
	// We iterate over all pixels in the layer, and initially set all unset neighbors to the first color in the gradient
	// if the current pixel is set on the original layer.
	// Then we iterate over all pixels again, and set each unset neighbor to the next color in the gradient if at least two
	// neighboring pixels are set on either layer with a different color.
	// NOTE: This could be optimized quite a bit.

	const minNeighbors = 3
	const numColors = 3

	bounds := layer.Bounds()
	effectLayer := image.NewRGBA(bounds)

	// Build a gradient index for all the colors we want to use.
	gradient := make([]color.Color, numColors)
	for i := 0; i < numColors; i++ {
		percentage := float64(i) / float64(numColors-1)
		gradient[i] = interpolateColor(colorA, colorB, percentage)
	}

	for colorIndex, c := range gradient {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				// Get the color of the current pixel on the original layer
				_, _, _, curPixelAlphaOrig := layer.At(x, y).RGBA()
				// If we have colorIndex 0, we set all unset neighbors to the first color in the gradient
				// if the current pixel is set on the original layer.
				if colorIndex == 0 {
					if curPixelAlphaOrig == 0 {
						continue
					}

					// Iterate over all neighbors and set them to the first color in the gradient if they are unset
					for i := -1; i <= 1; i++ {
						if x+i < bounds.Min.X || x+i >= bounds.Max.X {
							continue
						}
						for j := -1; j <= 1; j++ {
							if y+j < bounds.Min.Y || y+j >= bounds.Max.Y {
								continue
							}

							_, _, _, aNeighbor := layer.At(x+i, y+j).RGBA()
							if aNeighbor != 0 {
								continue
							}
							_, _, _, aEffectNeighbor := effectLayer.At(x+i, y+j).RGBA()
							if aEffectNeighbor == 0 {
								effectLayer.Set(x+i, y+j, gradient[0])
							}
						}
					}
				} else if curPixelAlphaOrig == 0 {
					// If the pixel is set on the original layer, we skip it since we don't want to cover it.

					// Make sure that the current pixel is set on the effect layer and
					// is not the current color (since we want to progress the gradient,
					// not stay on the same color).
					_, _, _, aEffect := effectLayer.At(x, y).RGBA()
					effectCol := effectLayer.At(x, y)
					if aEffect == 0 || effectCol == c {
						continue
					}

					// Check if at least two neighboring pixels are set on the effect layer
					// and are not the current color.
					var numNeighbors int
					for i := -1; i <= 1; i++ {
						if x+i < bounds.Min.X || x+i >= bounds.Max.X {
							continue
						}
						for j := -1; j <= 1; j++ {
							if y+j < bounds.Min.Y || y+j >= bounds.Max.Y {
								continue
							}

							// Make sure that the neighbor is set on the effect layer and
							// is not the current color.
							_, _, _, aEffectLeft := effectLayer.At(x+i, y+j).RGBA()
							effectLeft := effectLayer.At(x+i, y+j)
							if aEffectLeft != 0 && effectLeft != c {
								numNeighbors++
							}
						}
					}

					// If we don't have enough neighbors, we skip the current pixel.
					if numNeighbors < minNeighbors || rand.Float64() < 0.1 {
						continue
					}

					// Iterate over all neighbors and set them to the next color in the gradient if they are unset.
					for i := -1; i <= 1; i++ {
						if x+i < bounds.Min.X || x+i >= bounds.Max.X {
							continue
						}
						for j := -1; j <= 1; j++ {
							if y+j < bounds.Min.Y || y+j >= bounds.Max.Y {
								continue
							}

							// Set the neighbor to the next color in the gradient if it is unset.
							_, _, _, aEffectLeft := effectLayer.At(x+i, y+j).RGBA()
							if aEffectLeft == 0 {
								effectLayer.Set(x+i, y+j, c)
							}
						}
					}
				}
			}
		}
	}

	return effectLayer
}

// ApplyFlameEffect generates a flame effect for the given layer.
func ApplyFlameEffect(layer image.Image, colorA, colorB color.Color) *image.RGBA {
	return applyEffect(layer, colorA, colorB, 10, DirectionUp)
}

// ApplyDripEffect generates a drip effect for the given layer.
func ApplyDripEffect(layer image.Image, colorA, colorB color.Color) *image.RGBA {
	return applyEffect(layer, colorA, colorB, 15, DirectionDown)
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

// ApplyCorrosion generates a corrosion effect for the given layer.
// This will create splotches of a different color on the layer (wherever pixels are set)
// using cellular automata to grow the splotches from seedpoints selected at random.
// TODO:
// - Avoid eating into outlines.
// - We could 'chip away' at the corners of the sprites to make them look more worn.
func ApplyCorrosion(layer image.Image, color color.Color, numIterations, numSeeds int) *image.RGBA {
	bounds := layer.Bounds()

	// We will use two bool slices to represent the corrosion layer at two states.
	corrosionCur := make([]bool, bounds.Dx()*bounds.Dy())
	corrosionPrev := make([]bool, bounds.Dx()*bounds.Dy())

	// Iterate over the pixels in the original layer using rand.Perm, and set
	// the seed points for the corrosion layer if the pixel is set in the original layer.
	for _, i := range rand.Perm(bounds.Dx() * bounds.Dy()) {
		x := i % bounds.Dx()
		y := i / bounds.Dx()

		_, _, _, a := layer.At(x, y).RGBA()
		if a != 0 {
			corrosionCur[i] = true
			numSeeds--
		}

		if numSeeds == 0 {
			break
		}
	}

	// Iterate over the number of iterations
	for i := 0; i < numIterations; i++ {
		// Iterate over all cells in the layer
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				// If the current cell is not set in the original layer, skip it.
				// If there's nothing to corrode, we can skip the cell.
				_, _, _, a := layer.At(x, y).RGBA()
				if a == 0 {
					continue
				}

				// Check if the current cell is set in the previous state.
				// If so, copy it to the current state.
				index := y*bounds.Dx() + x
				if corrosionPrev[index] {
					corrosionCur[index] = true
					continue
				}

				// Check if any of the neighbors are set in the previous state.
				// The higher the number, the more likely the current cell will be set.
				var numNeighbors int
				for i := -1; i <= 1; i++ {
					if x+i < bounds.Min.X || x+i >= bounds.Max.X {
						continue
					}
					for j := -1; j <= 1; j++ {
						if y+j < bounds.Min.Y || y+j >= bounds.Max.Y {
							continue
						}

						if corrosionPrev[(y+j)*bounds.Dx()+x+i] {
							numNeighbors++
						}
					}
				}

				// Set the current cell based on the number of neighbors
				if rand.Intn(8) < numNeighbors {
					corrosionCur[index] = true
				}
			}
		}

		// Swap the current and previous corrosion states
		corrosionCur, corrosionPrev = corrosionPrev, corrosionCur
	}

	// Create a new RGBA image for the corrosion layer
	corrosionLayer := image.NewRGBA(bounds)

	// Iterate over all cells in the layer and set the color based on the corrosion state
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			index := y*bounds.Dx() + x
			if corrosionPrev[index] {
				corrosionLayer.Set(x, y, color)
			}
		}
	}

	return corrosionLayer
}
