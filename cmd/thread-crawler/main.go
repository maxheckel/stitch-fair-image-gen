package main

import (
	"database/sql"
	"fmt"
	"io"
	_ "modernc.org/sqlite"
	"net/http"
	"os"
	"strings"
)

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
