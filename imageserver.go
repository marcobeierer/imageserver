package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {

	imageserver := new(ImageServer)

	imageserver.Port = *flag.Int("port", 9999, "the port number of the imageserver")
	imageserver.ImagesPath = *flag.String("imagespath", "images", "path to the images directory")
	imageserver.CachePath = *flag.String("cachepath", "cache", "path to the cache directory")

	flag.Parse()

	imageserver.ImageFileTypes = []string{"jpg", "jpeg"} // TODO add support for png and gif
	imageserver.run()
}

type ImageServer struct {
	ImagesPath     string
	CachePath      string
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
		fmt.Fprint(writer, "invalid width value")

		return
	}

	height, err := getQueryIntValue(query, "height")
	if err != nil {

		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(writer, "invalid height value")

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
		fmt.Fprint(writer, "file not found")

		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {

		log.Printf("get file stats failed: %s\n", err.Error())

		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, "internal server error")

		return
	}

	data := []byte(fmt.Sprintf("%s%s%s%d", imagePath, fileInfo.Name(), fileInfo.Size(), fileInfo.ModTime().Unix()))
	hashValue := sha1.Sum(data)

	cacheImagePath := fmt.Sprintf("%s/%s/%x", is.CachePath, request.URL.Path[1:], hashValue)
	cacheImageFilePath := fmt.Sprintf("%s/%dx%d", cacheImagePath, width, height)

	_, err = os.Stat(cacheImageFilePath)
	if err == nil {

		http.ServeFile(writer, request, cacheImageFilePath)
		return
	}

	image, err := jpeg.Decode(file)
	if err != nil {

		log.Printf("decoding image failed: %s\n", err.Error())

		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, "internal server error")

		return
	}

	_, err = file.Seek(0, 0) // necessary for reading config
	if err != nil {

		log.Printf("seek failed: %s\n", err.Error())

		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, "internal server error")

		return
	}

	config, err := jpeg.DecodeConfig(file) // TODO could also use image instead of jpeg package
	if err != nil {

		log.Printf("decoding config failed: %s\n", err.Error())

		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, "internal server error")

		return
	}

	if width > uint(config.Width) || height > uint(config.Height) {

		writer.WriteHeader(http.StatusBadRequest) // TODO return more detailed error message or image at original size?
		fmt.Fprint(writer, "width and/or height value is greater than the original image dimensions")

		return
	}

	resizedImage := resize.Resize(width, height, image, resize.Lanczos3)

	err = os.MkdirAll(cacheImagePath, os.ModePerm)
	if err != nil {

		log.Printf("make cached file directory failed: %s\n", err.Error())

		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, "internal server error")

		return
	}

	fileInCache, err := os.Create(cacheImageFilePath)
	if err != nil {

		log.Printf("create cached file failed: %s\n", err.Error())

		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, "internal server error")

		return
	}
	defer fileInCache.Close()

	err = jpeg.Encode(fileInCache, resizedImage, nil)
	if err != nil {

		log.Printf("write cached file failed: %s\n", err.Error())

		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, "internal server error")

		return
	}

	http.ServeFile(writer, request, cacheImageFilePath)
	return
}
