package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/signintech/gopdf"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
	"image"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
)

type Annotator struct {
}

var DefaultAnnotator = Annotator{}

func ReadPdf(file multipart.File) ([]byte, error) {
	return DefaultAnnotator.ReadPdf(file)
}

func (a Annotator) ReadPdf(file multipart.File) ([]byte, error) {
	docBuff := new(bytes.Buffer)
	_, _ = io.Copy(docBuff, file)

	mimeDocument := mimetype.Detect(docBuff.Bytes())

	if !mimeDocument.Is("application/pdf") { //TODO buat converter dari jpg, doc k pdf
		log.Println(mimeDocument.String())
		if mimeDocument.Is("image/png") {

			data, err := convertPNGToPDF(file)

			if err != nil {
				log.Println(err)
				return nil, err
			}

			log.Println("masuk sini ges")

			//data, err := os.Open(pathPdf)
			pdfFile := bytes.NewReader(data)

			_, err = parseQrCode(pdfFile)
			return nil, nil
		}

		log.Println("file not supported1")

		return nil, errors.New("file not supported")

	}

	qr, err := parseQrCode(file)

	if err != nil {
		log.Println(err)
		panic(err)
	}

	return qr, nil

}

func parseQrCode(document io.ReadSeeker) ([]byte, error) {
	var (
		output = new(bytes.Buffer)
	)

	//fmt.Println("doc", document)
	_, totalPage, err := getPageInfo(document)

	//fmt.Println("1")
	if err != nil {
		fmt.Println("err", err)
		return nil, err
	}

	wmMap := make(map[int][]*pdfcpu.Watermark)

	drawQrCode()

	pathQr := "./assets/repo-qrcode.jpeg"
	qr, err := os.Open(pathQr)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer deleteFile(pathQr)
	defer qr.Close()
	qrfile, err := ioutil.ReadAll(qr)
	if err != nil {
		return nil, err
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

	log.Println("3")

	err = api.AddWatermarksSliceMap(document, output, wmMap, nil)

	if err != nil {
		return nil, err
	}

	filePath := "temp.pdf" // Path untuk file sementara
	fmt.Println("4")

	// Membuat file sementara untuk PDF
	tempFile, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer os.Remove(filePath) // Menghapus file sementara setelah selesai

	// Menyalin isi multipart.File ke file sementara
	_, err = io.Copy(tempFile, output)
	if err != nil {
		tempFile.Close()
		return nil, err
	}
	tempFile.Close()

	// Menambahkan properti ke file PDF menggunakan pdfcpu API
	properties := map[string]string{"Creator": "PT. Privy Identitas Digital", "Producer": "PrivyID PDF Processor"}

	path := "./images.pdf"
	err = api.AddPropertiesFile(filePath, path, properties, nil)
	if err != nil {
		return nil, err
	}

	defer deleteFile(path)

	log.Println("6")

	fileOpen, err := readFileToBytes(path)

	//err = ioutil.WriteFile("./imagea5.pdf", fileOpen, 0644)
	err = deleteFile(path)

	return fileOpen, err
}

func getPageInfo(document io.ReadSeeker) (pages map[int]MetaDocument, totalPage int, err error) {
	pages = make(map[int]MetaDocument)
	fmt.Println(",assao")
	ctxReader, err := api.ReadContext(document, nil)
	if err != nil {
		fmt.Println("siap111")
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

// Jika terdapat error, mohon install pdfcpu versi 0.3.13 ya dikarenakan penempelan PDFnya menggunakan
// versi 0.3.13. dibawah ini adalah cara installnya

// go get github.com/pdfcpu/pdfcpu@v0.3.13
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

func convertPNGToPDF(pngFile io.Reader) ([]byte, error) {
	images, _, err := image.Decode(pngFile)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println("siap")

	//Convert if image not 8 bit

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	err = pdf.ImageFrom(images, 0, 0, nil) //print image

	if err != nil {
		log.Println("print image got", err)
		return nil, err
	}
	pdfBytes := pdf.GetBytesPdf()
	//err = convertPNGToPDF(file, "./imagessss.pdf")
	if err != nil {

		log.Fatalf("Error converting PNG to PDF: %v", err)
		return nil, err
	}

	return pdfBytes, nil
}

func convertPNGToPDFv2(pngFile io.Reader, tempDir string) ([]byte, error) {

	outputPath := "./output.jpg"
	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	//imga, _, err := image.Decode(pngFile)
	//
	//if err != nil {
	//	fmt.Println("Error", err)
	//}
	//outputImage := resize.Resize(500, 500, imga, resize.Lanczos3)
	//
	//err = png.Encode(outputFile, outputImage)
	//
	//if err != nil {
	//	fmt.Printf("Error Convert Encode: %v", err)
	//	os.Exit(1)
	//}
	_, err = io.Copy(outputFile, pngFile)
	if err != nil {
		fmt.Printf("Error copying file contents: %v", err)
		os.Exit(1)
	}

	err = api.ImportImagesFile([]string{outputPath}, tempDir, nil, nil)
	//err := api.ImportImagesFile([]string{"31-2.jpg"}, "coba.pdf", nil, nil)

	if err != nil {
		fmt.Println("masuk rumah", err)
		panic(err)
		return nil, err
	}

	defer deleteFile(outputPath)

	if err != nil {
		return nil, err
	}

	return nil, err

}

type ByteReader struct {
	data []byte
	pos  int
}

func (r *ByteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func drawQrCode() {

	qrc, err := qrcode.New(fmt.Sprintf("https://privy.id/verify/%s", uuid.New()))
	if err != nil {
		fmt.Printf("could not generate QRCode: %v", err)
		return
	}

	imgLogo, err := os.Open("./assets/privy.png")

	if err != nil {
		log.Println(err)
		return
	}

	imageLogo, _, err := image.Decode(imgLogo)

	if err != nil {
		log.Println(err)
		return
	}

	w, err := standard.New("./assets/repo-qrcode.jpeg", standard.WithLogoImage(imageLogo), standard.WithQRWidth(10))
	if err != nil {
		fmt.Printf("standard.New failed: %v", err)
		return
	}

	// save file
	if err = qrc.Save(w); err != nil {
		fmt.Printf("could not save image: %v", err)
	}
}
