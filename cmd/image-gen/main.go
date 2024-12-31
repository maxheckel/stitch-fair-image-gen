package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/maxheckel/stitchfair-image-gen/src/util"
	"image"
	"image/color"
	"math"
	"math/rand"
	_ "modernc.org/sqlite"
	"os"
	"time"
)

type Config struct {
	ColorCount int
	Width      int
	Height     int
	Source     string
	DBPath     string
}

type Pixel struct {
	X   int    `json:"x"`
	Y   int    `json:"y"`
	Hex string `json:"hex"`
}

type Result struct {
	Base64PngImage string  `json:"base64PngImage"`
	Pixels         []Pixel `json:"pixels"`
}

func main() {

	config := configFromFlags()
	fmt.Println(config.Width, config.Height)

	// Open the input image
	inputFile, err := os.Open(config.Source)
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()
	source, _, err := image.Decode(inputFile)
	if err != nil {
		panic(err)
	}

	source = util.ResizeImage(source, config.Width, config.Height)
	source = util.TrimTransparent(source)

	threads := getThreadsFromDB(&config)

	newImage := ReplaceColors(source, threads, config.ColorCount)
	base64Str := util.EncodeImageToBase64(newImage)
	fmt.Println(base64Str)
	result := &Result{Base64PngImage: base64Str}
	for x := 0; x < newImage.Bounds().Dx(); x++ {
		for y := 0; y < newImage.Bounds().Dy(); y++ {

			r, g, b, a := newImage.At(x, y).RGBA()
			result.Pixels = append(result.Pixels, Pixel{
				X: x,
				Y: y,
				Hex: util.RGBAtoHex(color.RGBA{
					R: uint8(r),
					G: uint8(g),
					B: uint8(b),
					A: uint8(a),
				}),
			})
		}
	}
	resStr, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(resStr))
}

func configFromFlags() Config {
	config := Config{}
	var width = flag.Int("width", 5, "the width of the image")
	var height = flag.Int("height", 10, "the height of the image")
	var count = flag.Int("count", 5, "the color count count of the image")
	var imageSrc = flag.String("image_path", "./data/sample4.png", "the count of the image")
	var dbSrc = flag.String("thread_db_path", "./data/database.db", "the count of the image")
	flag.Parse()
	config.Width = *width
	config.Height = *height
	config.ColorCount = *count
	config.Source = *imageSrc
	config.DBPath = *dbSrc
	return config
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

func getThreadsFromDB(config *Config) []color.RGBA {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s", config.DBPath))

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
