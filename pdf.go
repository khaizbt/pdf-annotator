package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
)

func ReadPdf(file multipart.File, docHeader multipart.FileHeader) error {
	var (
		output = new(bytes.Buffer)
	)

	document, err := docHeader.Open()

	if err != nil {
		return err
	}
	fmt.Println("1")

	docBuff := new(bytes.Buffer)

	_, _ = io.Copy(docBuff, document)

	mimeDocument := mimetype.Detect(docBuff.Bytes())

	if !mimeDocument.Is("application/pdf") { //TODO buat converter dari jpg, doc k pdf
		return errors.New("document is not pdf")
	}

	//infoByts, totalPage, err :=
	_, totalPage, err := getPageInfo(document)

	if err != nil {
		return err
	}

	wmMap := make(map[int][]*pdfcpu.Watermark)

	qr, err := os.Open("./pkg/pdf/output.jpg")
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer qr.Close()
	qrfile, err := ioutil.ReadAll(qr)
	if err != nil {
		return err
	}
	fmt.Println("2")

	wmMap[totalPage] = append(wmMap[totalPage],
		createWatermark(
			qrfile,
			"qr",
			400,
			600,
			0.2,
		),
	)

	fmt.Println("3")

	err = api.AddWatermarksSliceMap(document, output, wmMap, nil)

	if err != nil {
		return err
	}

	filePath := "temp.pdf" // Path untuk file sementara
	fmt.Println("4")

	// Membuat file sementara untuk PDF
	tempFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer os.Remove(filePath) // Menghapus file sementara setelah selesai

	// Menyalin isi multipart.File ke file sementara
	_, err = io.Copy(tempFile, output)
	if err != nil {
		tempFile.Close()
		return err
	}
	tempFile.Close()

	fmt.Println("5")

	// Menambahkan properti ke file PDF menggunakan pdfcpu API
	properties := map[string]string{"Creator": "PT. Privy Identitas Digital", "Producer": "PrivyID PDF Processor"}

	path := "./images.pdf"
	err = api.AddPropertiesFile(filePath, path, properties, nil)
	if err != nil {
		return err
	}

	fmt.Println("6")

	fileOpen, err := readFileToBytes(path)

	err = ioutil.WriteFile("./imagea2.pdf", fileOpen, 0644)

	var pages []string
	pages = append(pages, "2")
	//get thumbnail
	err = api.ExtractImagesFile(filePath, "./thumbnail.png", pages, nil)

	if err != nil {
		return err
	}

	err = deleteFile(path)

	return err
}

func getPageInfo(document io.ReadSeeker) (pages map[int]MetaDocument, totalPage int, err error) {
	pages = make(map[int]MetaDocument)
	ctxReader, err := api.ReadContext(document, nil)
	if err != nil {
		return
	}

	media, err := ctxReader.PageBoundaries()
	if err != nil {
		return pages, totalPage, err
	}

	totalPage = len(media)

	for i, boundaries := range media {
		var (
			rotate      int
			orientation string
			w, h        float64
		)

		if boundaries.Media.Rect.Landscape() {
			orientation = "landscape"
		}
		if boundaries.Media.Rect.Portrait() {
			orientation = "portrait"
		}

		h = boundaries.Media.Rect.Height()
		w = boundaries.Media.Rect.Width()

		rotate = boundaries.Rot

		if rotate == 90 || rotate == 270 {
			w, h = h, w
		}
		pages[i+1] = MetaDocument{
			Page:        int64(i + 1),
			Width:       w,
			Height:      h,
			Orientation: orientation,
			Rotate:      rotate,
		}
	}

	return
}

// TODO buat function tempel PDF
func createWatermark(img []byte, typeWatermark string, x, y int, scale float64) *pdfcpu.Watermark {
	wm := pdfcpu.DefaultWatermarkConfig()
	wm.Mode = pdfcpu.WMImage
	wm.Image = bytes.NewReader(img)
	wm.Pos = pdfcpu.TopLeft
	wm.Update = false
	wm.OnTop = true
	wm.Dx = x
	wm.Dy = y - (y * 2)
	//wm.Dy = y
	wm.Scale = scale
	wm.Rotation = 0
	wm.Diagonal = pdfcpu.NoDiagonal
	if typeWatermark == "qr" {
		wm.Pos = pdfcpu.BottomRight
		wm.Dx = -10
		wm.Dy = 10
	}

	wm.ScaleAbs = true

	return wm
}
func readFileToBytes(filePath string) ([]byte, error) {
	// Baca file ke dalam bytes
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

func deleteFile(filePath string) error {
	// Hapus file
	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}

//func generateThumbnailFromPDF(inputPath, thumbnailPath string, pageNumber int) error {
//	// Read the PDF using pdfcpu API
//	ctx, err := api.ReadContextFile(inputPath)
//	if err != nil {
//		return err
//	}
//
//	// Extract images from the specified page
//	err := api.Ext(ctx, int(1), nil)
//	if err != nil {
//		return err
//	}
//
//	// Select the first image from the extracted images
//	if len(imageList) == 0 {
//		return fmt.Errorf("no images found on page %d", pageNumber)
//	}
//	selectedImage := imageList[0]
//
//	// Decode the selected image
//	img, _, err := image.Decode(selectedImage)
//	if err != nil {
//		return err
//	}
//
//	// Create the thumbnail image with the desired size
//	thumbnail := resizeImage(img, 200, 200)
//
//	// Save the thumbnail as JPEG
//	file, err := os.Create(thumbnailPath)
//	if err != nil {
//		return err
//	}
//	defer file.Close()
//
//	err = jpeg.Encode(file, thumbnail, &jpeg.Options{Quality: 90})
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func resizeImage(img image.Image, width, height int) image.Image {
//	thumbnail := image.NewRGBA(image.Rect(0, 0, width, height))
//	g := resize.Thumbnail(uint(width), uint(height), img, resize.Lanczos3)
//	draw.Draw(thumbnail, thumbnail.Bounds(), g, image.Point{}, draw.Src)
//	return thumbnail
//}
