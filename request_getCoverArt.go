package main

import (
	"database/sql"
	"image"
	"image/jpeg"
	"os"
	"path"
	"strconv"
	"github.com/nfnt/resize"
	_ "image/png"
	_ "github.com/mxk/go-sqlite/sqlite3"
)

func (connection *ClientConnection) getCoverArt() {
	id,error := strconv.Atoi( connection.parameters["id"] )
	if error != nil {
		connection.responseError( 10 )
		return
	}
	db,error := sql.Open( "sqlite3", settings.Database )
	if error != nil {
		println( "Error opening database", error )
		connection.responseError( 0 )
		return
	}
	defer db.Close()
	songRow := db.QueryRow( "SELECT songParent FROM songs WHERE songKey=?;", id )
	var songParent int
	error = songRow.Scan( &songParent )
	if error == nil {
		id = songParent
	}
	folderRow := db.QueryRow( "SELECT folderPath,folderRoot FROM folders WHERE folderKey=?;", id )
	var folderPath string
	var folderRoot string
	error = folderRow.Scan( &folderPath, &folderRoot )
	if error != nil {
		println( "Error reading cover art folder from database", error )
		connection.responseError( 70 )
		return
	}
	coverFilePath := path.Join( folderRoot, folderPath, "cover.jpg" )
	coverFile,error := os.Open( coverFilePath )
	if error != nil {
		coverFilePath = path.Join( folderRoot, folderPath, "cover.png" )
		coverFile,error = os.Open( coverFilePath )
		if error != nil {
			println( "Unable to open cover art file (jpg or png)", folderPath, error.Error() )
			connection.responseError( 70 )
			return
		}
	}
	defer coverFile.Close()

	originalImage,_,error := image.Decode( coverFile )
	if error != nil {
		println( "Error decoding cover art", error.Error() )
		connection.responseError( 70 )
		return
	}
	originalWidth := originalImage.Bounds().Dx()
	originalHeight := originalImage.Bounds().Dy()
	var resizedImage image.Image
	imageSize,error := strconv.Atoi( connection.parameters["size"] )
	if error != nil || (imageSize > originalWidth && imageSize > originalHeight) {
		resizedImage = originalImage
	} else {
		resizedImage = resize.Resize( uint(imageSize), uint(imageSize), originalImage, resize.Bicubic )
	}
	jpegOptions := jpeg.Options{ 80 }
	jpeg.Encode( connection.output, resizedImage, &jpegOptions )
}
