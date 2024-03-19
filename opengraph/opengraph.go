package opengraph

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"golang.org/x/net/html"
)

type OpenGraph struct {
	Client *http.Client
}

// FetchMetaTags fetches meta tags from the head section of a given URL
func (og *OpenGraph) FetchMetaTags(ctx context.Context, targetURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := og.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	var metaTags []string
	tokenizer := html.NewTokenizer(resp.Body)
	var inHead bool
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				return metaTags, nil // EOF is expected
			}
			return nil, fmt.Errorf("reading body: %w", err)
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "head" {
				inHead = true
			}
			if inHead && token.Data == "meta" {
				metaTags = append(metaTags, token.String())
			}
		case html.EndTagToken:
			token := tokenizer.Token()
			if inHead && token.Data == "head" {
				return metaTags, nil // End of head section
			}
		}
	}
}

func FetchMetaTags(ctx context.Context, targetURL string) ([]string, error) {
	og := OpenGraph{Client: &http.Client{
		Timeout: 2 * time.Second,
	}}
	return og.FetchMetaTags(ctx, targetURL)
}

// // CaptureScreenshot takes a context, URL, captures a screenshot of the webpage,
// // and returns the base64 encoded image.
// func CaptureScreenshot(ctx context.Context, url string) (string, error) {
// 	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
// 	defer cancel()

// 	var buf []byte
// 	if err := chromedp.Run(ctx,
// 		chromedp.Navigate(url),
// 		chromedp.CaptureScreenshot(&buf),
// 	); err != nil {
// 		return "", err
// 	}

// 	base64Img := base64.StdEncoding.EncodeToString(buf)

// 	file, err := os.Create("screenshot.png")
// 	if err != nil {
// 		return "", err
// 	}
// 	defer file.Close()

// 	if _, err := io.Copy(file, bytes.NewReader(buf)); err != nil {
// 		return "", err
// 	}

// 	return base64Img, nil
// }

func EncodeImageToBase64(img image.Image) string {
	// Encode the image as PNG
	buf := new(bytes.Buffer)
	_ = png.Encode(buf, img)

	// Encode the PNG image as base64
	encodedStr := base64.StdEncoding.EncodeToString(buf.Bytes())

	return encodedStr
}
func GenerateBarcode(url string) (image.Image, error) {
	qrCode, err := qr.Encode(url, qr.L, qr.Auto)
	if err != nil {
		return nil, err
	}
	qrCode, err = barcode.Scale(qrCode, 200, 200)
	if err != nil {
		return nil, err
	}
	return qrCode, nil
}

func SaveBarcodeToFile(barcode image.Image, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = png.Encode(file, barcode)
	if err != nil {
		return err
	}
	return nil
}
