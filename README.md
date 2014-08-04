# Image Server

## Instruction
The image server is currently able to resize images and deliver them on the fly. The following request would for example resize an image to a width of 500px and deliver the resized image to the client.

	http://localhost:9999/path/to/image.jpg?width=500

## Build and Run from Source
	go get github.com/webguerilla/imageserver
	mkdir images
	mkdir cache
	go build
	./imageserver

## Warning
The image server is currently vulnerable to DoS attacks and thus not qualified for production use. The cause of that issue is that an attacker could request a high number of width/height combinations for each image and the generation of the resized images is quite expensive. The cached files will also take up some disk space.
