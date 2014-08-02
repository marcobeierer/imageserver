package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
	"image/jpeg"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {

	imageserver := new(ImageServer)

	imageserver.Port = 9999
	imageserver.ImagesPath = "images"
	imageserver.ImageFileTypes = []string{"jpg", "jpeg"}

	imageserver.run()
}

type ImageServer struct {
	ImagesPath     string
	Port           int
	ImageFileTypes []string
}

func (is *ImageServer) run() {

	router := mux.NewRouter()

	filetypes := strings.Join(is.ImageFileTypes, "|")
	router.HandleFunc("/{path:[a-zA-Z0-9\\-_\\/]+\\.("+filetypes+")}", is.ImageHandler)

	http.Handle("/", router)
	http.ListenAndServe(fmt.Sprintf(":%d", is.Port), nil)

}

func getQueryIntValue(query map[string][]string, key string) (uint, error) {

	valueSlice, ok := query[key]
	if ok {

		value, err := strconv.ParseUint(valueSlice[0], 10, 0)
		if err != nil {
			return 0, err
		}

		return uint(value), nil
	}

	return 0, nil
}

func (is *ImageServer) ImageHandler(writer http.ResponseWriter, request *http.Request) {

	query := request.URL.Query()

	width, err := getQueryIntValue(query, "width")
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	height, err := getQueryIntValue(query, "height")
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	imagePath := fmt.Sprintf("%s/%s", is.ImagesPath, request.URL.Path[1:])

	if width == 0 && height == 0 {

		http.ServeFile(writer, request, imagePath)
		return
	}

	file, err := os.Open(imagePath)
	if err != nil {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	image, err := jpeg.Decode(file)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
	}
	file.Close()

	resizedImage := resize.Resize(width, height, image, resize.Lanczos3)
	jpeg.Encode(writer, resizedImage, nil)
}
