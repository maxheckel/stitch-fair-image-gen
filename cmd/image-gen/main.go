package main

import (
	"database/sql"
	"github.com/maxheckel/stitchfair-image-gen/src/util"
	"image"
	"image/color"
	"math"
	"math/rand"
	_ "modernc.org/sqlite"
	"os"
	"sort"
	"time"
)

type Config struct {
	ColorCount int
	Width      int
	Height     int
}

type ColorReplacement struct {
	OriginalColor util.RGBA
	NewColor      util.RGBA
}

func main() {
	// Open the input image
	inputFile, err := os.Open("./data/sample4.png")
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()

	config := Config{
		ColorCount: 489,
		Width:      10,
		Height:     10,
	}

	source, _, err := image.Decode(inputFile)
	if err != nil {
		panic(err)
	}

	source = util.ResizeImage(source, config.Width, config.Height)
	source = util.TrimTransparent(source)

	threads := getThreadsFromDB(err)

	newImage := ReplaceColors(source, threads, config.ColorCount)
	err = util.SaveImage(newImage, "./data/output.png", "png")
	if err != nil {
		panic(err)
	}
}

// ReplaceColors replaces the colors in the image with the closest matches from the given color palette.
// Transparent pixels are ignored.
func ReplaceColors(img image.Image, palette []color.RGBA, targetCount int) image.Image {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	chunkSize := len(palette) / (targetCount - 2)
	targetColors := []color.RGBA{}
	targetColors = append(targetColors, findClosestColor(util.GetDarkestPixel(img), palette))
	targetColors = append(targetColors, findClosestColor(util.GetBrightestPixel(img), palette))
	for x := 0; x < len(palette); x += chunkSize {
		end := x + chunkSize
		if end > len(palette) {
			end = len(palette)
		}
		s := rand.NewSource(time.Now().Unix())
		r := rand.New(s) // initialize local pseudorandom generator
		targetColors = append(targetColors, palette[x:end][r.Intn(end-x)])
	}
	// Replace each pixel with the closest color from the target colors
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a == 0 { // Ignore transparent pixels
				result.Set(x, y, color.RGBA{0, 0, 0, 0}) // Preserve transparency
				continue
			}

			original := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
			closest := findClosestColor(original, targetColors)
			result.Set(x, y, closest)
		}
	}

	return result
}

// selectTargetColors selects the most representative colors from the palette based on image usage.
// Transparent pixels are ignored.
func selectTargetColor(img image.Image, palette []color.RGBA) color.RGBA {
	colorUsage := map[color.RGBA]int{}
	bounds := img.Bounds()

	// Count the occurrence of each color in the image, ignoring transparency
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a == 0 { // Ignore transparent pixels
				continue
			}

			c := color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
			colorUsage[c]++
		}
	}

	// Rank the palette colors based on their closeness to the image's colors
	var colorScores []struct {
		color color.RGBA
		score int
	}
	for _, p := range palette {
		score := 0
		for imgColor, _ := range colorUsage {
			score += int(colorDistance(imgColor, p))
		}
		colorScores = append(colorScores, struct {
			color color.RGBA
			score int
		}{p, score})
	}

	// Sort palette colors by their scores
	sort.Slice(colorScores, func(i, j int) bool {
		return colorScores[i].score < colorScores[j].score
	})

	return colorScores[0].color
}

// findClosestColor finds the closest color in the palette to the given color.
func findClosestColor(c color.RGBA, palette []color.RGBA) color.RGBA {
	minDistance := math.MaxFloat64
	var closest color.RGBA

	for _, p := range palette {
		d := colorDistance(c, p)
		if d < minDistance {
			minDistance = d
			closest = p
		}
	}

	return closest
}

// colorDistance calculates the Euclidean distance between two colors in RGBA space.
func colorDistance(c1, c2 color.RGBA) float64 {
	rDiff := int(c1.R) - int(c2.R)
	gDiff := int(c1.G) - int(c2.G)
	bDiff := int(c1.B) - int(c2.B)
	aDiff := int(c1.A) - int(c2.A)
	return math.Sqrt(float64(rDiff*rDiff + gDiff*gDiff + bDiff*bDiff + aDiff*aDiff))
}

func getThreadsFromDB(err error) []color.RGBA {
	db, err := sql.Open("sqlite", "file:./data/database.db")

	if err != nil {
		panic(err)
	}
	defer db.Close()
	threads := []color.RGBA{}
	colors, err := db.Query("select hex from threads order by hue_order")
	if err != nil {
		panic(err)
	}
	for colors.Next() {
		var hex string
		err = colors.Scan(&hex)
		if err != nil {
			panic(err)
		}
		color := color.RGBA{}
		color.R, color.G, color.B, err = util.HexToRGB(hex)
		color.A = 255
		threads = append(threads, color)
	}
	return threads
}
