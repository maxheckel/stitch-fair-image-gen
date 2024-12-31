package util

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strconv"
	"strings"
)

func ImgToBase64(img image.Image) {

}

func GetDarkestPixel(img image.Image) (darkest color.RGBA) {
	bounds := img.Bounds()
	minLuminance := float64(1<<16 - 1) // Initialize with the maximum luminance value.

	for i := 0; i < bounds.Dx(); i++ {
		for j := 0; j < bounds.Dy(); j++ {
			pixel := img.At(i, j)
			r, g, b, _ := pixel.RGBA()

			// Calculate luminance (approximation of human perception)
			luminance := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)

			if luminance < minLuminance {
				minLuminance = luminance
				darkest = color.RGBA{
					R: uint8(r),
					G: uint8(g),
					B: uint8(b),
					A: 0,
				}
			}
		}
	}

	return darkest
}

func GetBrightestPixel(img image.Image) (lightest color.RGBA) {
	bounds := img.Bounds()
	maxLuminance := float64(0) // Initialize with the maximum luminance value.

	for i := 0; i < bounds.Dx(); i++ {
		for j := 0; j < bounds.Dy(); j++ {
			pixel := img.At(i, j)
			r, g, b, _ := pixel.RGBA()

			// Calculate luminance (approximation of human perception)
			luminance := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)

			if luminance > maxLuminance {
				maxLuminance = luminance
				lightest = color.RGBA{
					R: uint8(r),
					G: uint8(g),
					B: uint8(b),
					A: 0,
				}
			}
		}
	}

	return lightest
}

// ResizeImage resizes an image to the specified width and height.
func ResizeImage(img image.Image, width, height int) image.Image {
	// Create a new blank image with the desired dimensions
	resizedImg := image.NewRGBA(image.Rect(0, 0, width, height))

	// Use the draw package to scale the image
	draw.BiLinear.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	return resizedImg
}

// FindClosestColorInPalette finds the closest color in the palette to a given color.
func FindClosestColorInPalette(c color.RGBA, palette []color.RGBA) color.RGBA {
	minDistance := float64(^uint(0))
	var closest color.RGBA

	for _, p := range palette {
		dist := colorDistance(c, p)
		if dist < minDistance {
			minDistance = dist
			closest = p
		}
	}

	return closest
}

// colorDistance calculates the squared distance between two colors in RGB space.
func colorDistance(c1, c2 color.RGBA) float64 {
	dr := float64(c1.R - c2.R)
	dg := float64(c1.G - c2.G)
	db := float64(c1.B - c2.B)
	da := float64(c1.A - c2.A)
	return dr*dr + dg*dg + db*db + da*da
}

// EncodeImageToBase64 takes an image and encodes it to a base64 string
func EncodeImageToBase64(img image.Image) string {
	// Create a buffer to store the encoded image
	buf := new(bytes.Buffer)

	// Encode the image to PNG format
	err := png.Encode(buf, img) // Use other encoders like jpeg.Encode if needed
	if err != nil {
		panic(err)
	}

	// Encode the buffer to base64
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func RGBAtoHex(c color.RGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// SaveImage saves an image to a file in the specified format.
// Supported formats: "png", "jpeg"
func SaveImage(img image.Image, filename, format string) error {
	// Create the output file
	outputFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Encode the image in the specified format
	switch format {
	case "png":
		err = png.Encode(outputFile, img)
	case "jpeg", "jpg":
		// Use default options for JPEG encoding
		err = jpeg.Encode(outputFile, img, nil)
	default:
		return errors.New("unsupported image format: " + format)
	}

	return err
}

// TrimTransparent trims transparent pixels from an image and returns the resulting image.
func TrimTransparent(img image.Image) image.Image {
	bounds := img.Bounds()

	// Determine the new bounds by inspecting non-transparent pixels
	var minX, minY, maxX, maxY int
	found := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 { // Check alpha channel
				if !found {
					minX, minY = x, y
					maxX, maxY = x, y
					found = true
				} else {
					if x < minX {
						minX = x
					}
					if y < minY {
						minY = y
					}
					if x > maxX {
						maxX = x
					}
					if y > maxY {
						maxY = y
					}
				}
			}
		}
	}

	if !found {
		// If no non-transparent pixel is found, return an empty image
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}

	// Create a new image with the calculated bounds
	newBounds := image.Rect(0, 0, maxX-minX+1, maxY-minY+1)
	trimmedImg := image.NewRGBA(newBounds)

	// Copy the relevant part of the original image to the new image
	draw.Draw(trimmedImg, newBounds, img, image.Point{X: minX, Y: minY}, draw.Src)

	return trimmedImg
}

// HexToRGB converts a hex color string to RGB values.
func HexToRGB(hex string) (uint8, uint8, uint8, error) {
	// Remove the leading '#' if it exists
	hex = strings.TrimPrefix(hex, "#")

	// Ensure the hex string is valid
	if len(hex) != 6 {
		return 0, 0, 0, errors.New("invalid hex color format")
	}

	// Parse the R, G, and B components
	r, err := strconv.ParseInt(hex[0:2], 16, 0)
	if err != nil {
		return 0, 0, 0, err
	}

	g, err := strconv.ParseInt(hex[2:4], 16, 0)
	if err != nil {
		return 0, 0, 0, err
	}

	b, err := strconv.ParseInt(hex[4:6], 16, 0)
	if err != nil {
		return 0, 0, 0, err
	}

	return uint8(r), uint8(g), uint8(b), nil
}
