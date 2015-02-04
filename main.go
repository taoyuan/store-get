package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
)

var storeUrl, _ = url.Parse("https://search.apps.ubuntu.com/api/v1/package/")

type storePayload struct {
	AnonDownloadUrl string `json:"anon_download_url"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage:", os.Args[0], "[snappy package]")
		os.Exit(1)
	}

	pkgName := os.Args[1]

	if !strings.Contains(pkgName, ".") {
		pkgName = "com.ubuntu.snappy." + pkgName
	}

	u, err := storeUrl.Parse(pkgName)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("Fetching package information from", u.String())
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	req.Header.Set("Accept", "application/hal+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error when talking to server:", resp.StatusCode)
		os.Exit(1)
	}

	var payload storePayload
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&payload); err != nil {
		fmt.Println("Error: cannot obtain download url from store:", err)
		os.Exit(1)
	}

	if payload.AnonDownloadUrl == "" {
		fmt.Println("Error: no download link for package")
		os.Exit(1)
	}

	downloadPath := path.Base(payload.AnonDownloadUrl)
	downloadPath = strings.TrimPrefix(downloadPath, "com.ubuntu.snappy.")

	f, err := os.Create(downloadPath)
	if err != nil {
		fmt.Println("Error: cannot create target to download to")
		os.Exit(1)
	}
	defer f.Close()

	fmt.Println("Downloading", payload.AnonDownloadUrl)
	resp, err = http.Get(payload.AnonDownloadUrl)
	if err != nil {
		fmt.Println("Error: cannot download package:", err)
		os.Exit(1)
	}

	sourceSize, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

	bar := pb.New(int(sourceSize)).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10)
	bar.ShowSpeed = true
	bar.Start()
	defer bar.Finish()

	writer := io.MultiWriter(f, bar)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		fmt.Println("Error: cannot download package:", err)
		os.Remove(downloadPath)
		os.Exit(1)
	}

	fmt.Println("Snappy package downloaded to", downloadPath)
}
