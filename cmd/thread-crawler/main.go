package main

import (
	"database/sql"
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"math"
	_ "modernc.org/sqlite"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Thread struct {
	Name      string
	ImagePath string
	URL       string
	Hex       string
}

func main() {
	res, err := os.ReadFile("data/threads.txt")
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("sqlite", "file:./data/database.db")

	if err != nil {
		panic(err)
	}
	defer db.Close()

	lines := strings.Split(string(res), "\n")
	go downloadImages(lines)
	crawlURLsAndWriteThreads(lines, db)
	addAverageColorToThreads(err, db)

	rows, err := db.Query("select hex from threads")
	hexes := []string{}
	for rows.Next() {
		var hex string
		err = rows.Scan(&hex)
		if err != nil {
			panic(err)
		}
		hexes = append(hexes, hex)
	}
	sortedHexes, err := SortHexByHue(hexes)
	if err != nil {
		panic(err)
	}
	for index, hex := range sortedHexes {
		_, err = db.Exec("update threads set hue_order = ? where hex = ?", index, hex)
		if err != nil {
			panic(err)
		}
	}
}

func addAverageColorToThreads(err error, db *sql.DB) {
	colors, err := db.Query("SELECT name, image_path, url, hex FROM threads where hex is null or hex = ''")
	if err != nil {
		panic(err)
	}
	threadsToWrite := []*Thread{{}}
	for colors.Next() {
		thread := &Thread{}
		err = colors.Scan(&thread.Name, &thread.ImagePath, &thread.URL, &thread.Hex)
		if err != nil {
			panic(err)
		}

		img, err := decodeImage(thread.ImagePath)

		if err != nil {
			panic(err)
		}

		var rTotal, gTotal, bTotal uint32
		for x := 0; x < img.Bounds().Dx(); x++ {
			for y := 0; y < img.Bounds().Dy(); y++ {
				r, g, b, _ := img.At(x, y).RGBA()
				// Divide by 257 because these RGB elements are preconfigured for
				// alpha with 1 to #ffff colors. Some 32 bit int tomfoolery
				rTotal += r / 257
				gTotal += g / 257
				bTotal += b / 257
			}
		}
		pixels := uint32(img.Bounds().Dy() * img.Bounds().Dx())
		rAverage := rTotal / pixels
		gAverage := gTotal / pixels
		bAverage := bTotal / pixels

		thread.Hex = fmt.Sprintf("#%02x%02x%02x", rAverage, gAverage, bAverage)
		threadsToWrite = append(threadsToWrite, thread)
	}

	for _, thread := range threadsToWrite {
		err = updateThread(thread, db)
		if err != nil {
			panic(err)
		}
	}
}

func updateThread(thread *Thread, db *sql.DB) error {
	_, err := db.Exec("update threads set hex = ? where name = ?", thread.Hex, thread.Name)
	return err
}

func decodeImage(filename string) (image.Image, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}
func crawlURLsAndWriteThreads(lines []string, db *sql.DB) {
	for _, line := range lines {
		lineData := strings.Split(line, " /images/")
		color, imagePath := lineData[0], lineData[1]
		imagePath = "/images/" + imagePath
		filename := "data/color-swatches/" + color + ".jpeg"
		thread := &Thread{
			ImagePath: filename,
			Name:      color,
		}

		if threadExists(thread, db) {
			fmt.Println("Thread already exists, skipping")
			continue
		}

		var r io.Reader = strings.NewReader(fmt.Sprintf("{ SearchTerm: \"%s\" }", color))
		lookup, err := http.Post("https://www.everythingcrossstitch.com/mrsfService.asmx/SearchSite", "application/json", r)
		if err != nil {
			panic(err)
		}

		body, err := io.ReadAll(lookup.Body)
		if err != nil {
			panic(err)
		}
		// Regular expression to match the <entitykey> tag content
		re := regexp.MustCompile(`<entitykey>(.*?)</entitykey>`)

		// Find the first match
		match := re.FindStringSubmatch(string(body))
		var url string
		if len(match) > 1 {
			url = fmt.Sprintf("https://www.everythingcrossstitch.com/-mrp-%s.aspx", match[1])
		}
		thread.URL = url
		err = writeThread(thread, db)
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("Wrote thread %s", thread.Name))
	}
}

func threadExists(thread *Thread, db *sql.DB) bool {
	var exists string
	err := db.QueryRow("SELECT name FROM threads where name = ?", thread.Name).Scan(&exists)
	if err == sql.ErrNoRows {
		return false
	}
	return true
}

func writeThread(thread *Thread, db *sql.DB) error {
	if threadExists(thread, db) {
		return nil
	}
	_, err := db.Exec("INSERT INTO threads (name, image_path, hex, url) values (?, ?, ?, ?)", thread.Name, thread.ImagePath, thread.Hex, thread.URL)
	if err != nil {
		return err
	}
	return nil
}

func downloadImages(lines []string) {
	for _, line := range lines {
		lineData := strings.Split(line, " /images/")
		color, imagePath := lineData[0], lineData[1]
		imagePath = "/images/" + imagePath
		filename := "data/color-swatches/" + color + ".jpeg"
		if _, err := os.Stat(filename); err == nil {
			continue
		}
		res, err := http.Get(fmt.Sprintf("https://www.everythingcrossstitch.com%s", imagePath))
		if err != nil {
			panic(err)
		}
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(filename, bodyBytes, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
}

// HexToRGB converts a hex color code to RGB values.
func HexToRGB(hex string) (r, g, b uint8, err error) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, fmt.Errorf("invalid hex code: %s", hex)
	}
	rInt, err := strconv.ParseUint(hex[1:3], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	gInt, err := strconv.ParseUint(hex[3:5], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	bInt, err := strconv.ParseUint(hex[5:7], 16, 8)
	if err != nil {
		return 0, 0, 0, err
	}
	return uint8(rInt), uint8(gInt), uint8(bInt), nil
}

// RGBToHue converts RGB values to a hue angle (in degrees).
func RGBToHue(r, g, b uint8) float64 {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))

	var h float64
	if max == min {
		h = 0 // Gray scale
	} else if max == rf {
		h = 60 * (((gf - bf) / (max - min)) + 6) // +6 to ensure positive values
	} else if max == gf {
		h = 60 * (((bf - rf) / (max - min)) + 2)
	} else if max == bf {
		h = 60 * (((rf - gf) / (max - min)) + 4)
	}
	return math.Mod(h, 360) // Normalize hue to 0–360 degrees
}

// SortHexByHue sorts a list of hex color codes from red to violet based on hue.
func SortHexByHue(hexCodes []string) ([]string, error) {
	colors := make([]struct {
		Hex string
		Hue float64
	}, len(hexCodes))

	for i, hex := range hexCodes {
		r, g, b, err := HexToRGB(hex)
		if err != nil {
			return nil, err
		}
		colors[i] = struct {
			Hex string
			Hue float64
		}{
			Hex: hex,
			Hue: RGBToHue(r, g, b),
		}
	}

	sort.Slice(colors, func(i, j int) bool {
		return colors[i].Hue < colors[j].Hue
	})

	sortedHex := make([]string, len(hexCodes))
	for i, c := range colors {
		sortedHex[i] = c.Hex
	}
	return sortedHex, nil
}
