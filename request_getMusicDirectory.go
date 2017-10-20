package main

import (
	"database/sql"
	"os"
	"path"
	"strconv"
	"time"
	_ "github.com/mxk/go-sqlite/sqlite3"
)

func (connection *ClientConnection) getMusicDirectory() {
	var content string
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
	parentRow := db.QueryRow( "SELECT folderName,folderParent FROM folders WHERE folderKey=?;", id )
	var parentName string
	var parentParent int
	error = parentRow.Scan( &parentName, &parentParent )
	if error != nil {
		println( "Error reading folder name from database", error )
		connection.responseError( 70 )
		return
	}
	if parentParent != 0 {
		content = `<directory id="` + strconv.Itoa(id) + `" parent="` + strconv.Itoa(parentParent) + `" name="` + xmlString(parentName) + `">`
	} else {
		content = `<directory id="` + strconv.Itoa(id) + `" name="` + xmlString(parentName) + `">`
	}

	// List any child folders
	childFolderRows,error := db.Query( "SELECT folderKey,folderName,folderPath,folderCreated,folderRoot FROM folders WHERE folderParent=? ORDER BY folderName;", id )
	if error != nil {
		println( "Error reading folders from database", error )
		connection.responseError( 0 )
		return
	}
	for childFolderRows.Next() {
		var folderKey int
		var folderName string
		var folderPath string
		var folderCreated int64
		var folderRoot string
		childFolderRows.Scan( &folderKey, &folderName, &folderPath, &folderCreated, &folderRoot )
		timestamp := time.Unix( folderCreated, 0 )
		hasCoverArt := true
		coverArtPath := path.Join( folderRoot, folderPath, "cover.jpg" )
		if _,error := os.Stat( coverArtPath ); error != nil {
			coverArtPath = path.Join( folderRoot, folderPath, "cover.png" )
			if _,error := os.Stat( coverArtPath ); error != nil {
				hasCoverArt = false
			}
		}
		if hasCoverArt {
			content += `<child id="` + strconv.Itoa(folderKey) + `" parent="` + strconv.Itoa(id) + `" title="` + xmlString(folderName) + `" album="` + xmlString(folderName) + `" artist="` + xmlString(parentName) + `" isDir="true" coverArt="` + strconv.Itoa(folderKey) + `" created="` + timestamp.Format( time.RFC3339 ) + `"/>`
		} else {
			content += `<child id="` + strconv.Itoa(folderKey) + `" parent="` + strconv.Itoa(id) + `" title="` + xmlString(folderName) + `" album="` + xmlString(folderName) + `" artist="` + xmlString(parentName) + `" isDir="true" created="` + timestamp.Format( time.RFC3339 ) + `"/>`
		}
	}

	// List any child songs
	childSongRows,error := db.Query( "SELECT songKey,songPath,songCreated,songTitle,songArtist,songAlbum,songDuration,songYear,songTrack,songDisc,folderPath,folderRoot FROM songs,folders WHERE songParent=? AND folderKey=songParent ORDER BY songDisc,songTrack,songTitle;", id )
	if error != nil {
		println( "Error reading songs from database", error )
		connection.responseError( 0 )
		return
	}
	for childSongRows.Next() {
		var songKey int
		var songPath string
		var songCreated int64
		var songTitle string
		var songArtist string
		var songAlbum string
		var songDuration int
		var songYear int
		var songTrack int
		var songDisc int
		var folderPath string
		var folderRoot string
		childSongRows.Scan( &songKey, &songPath, &songCreated, &songTitle, &songArtist, &songAlbum, &songDuration, &songYear, &songTrack, &songDisc, &folderPath, &folderRoot )
		timestamp := time.Unix( songCreated, 0 )
		hasCoverArt := true
		coverArtPath := path.Join( folderRoot, folderPath, "cover.jpg" )
		if _,error := os.Stat( coverArtPath ); error != nil {
			coverArtPath = path.Join( folderRoot, folderPath, "cover.png" )
			if _,error := os.Stat( coverArtPath ); error != nil {
				hasCoverArt = false
			}
		}
		if hasCoverArt {
			content += `<child id="` + strconv.Itoa(songKey) + `" parent="` + strconv.Itoa(id) + `" title="` + xmlString(songTitle) + `" album="` + xmlString(songAlbum) + `" artist="` + xmlString(songArtist) + `" isDir="false" coverArt="` + strconv.Itoa(id) + `" created="` + timestamp.Format( time.RFC3339 ) + `" duration="` + strconv.Itoa(songDuration) + `" track="` + strconv.Itoa(songTrack) + `" discNumber="` + strconv.Itoa(songDisc) + `" year="` + strconv.Itoa(songYear) + `" suffix="m4a" contentType="audio/mp4" isVideo="false" path="` + xmlString(songPath) + `" type="music" transcodedSuffix="mp3" transcodedContentType="audio/mpeg"/>`
		} else {
			content += `<child id="` + strconv.Itoa(songKey) + `" parent="` + strconv.Itoa(id) + `" title="` + xmlString(songTitle) + `" album="` + xmlString(songAlbum) + `" artist="` + xmlString(songArtist) + `" isDir="false" created="` + timestamp.Format( time.RFC3339 ) + `" duration="` + strconv.Itoa(songDuration) + `" track="` + strconv.Itoa(songTrack) + `" discNumber="` + strconv.Itoa(songDisc) + `" year="` + strconv.Itoa(songYear) + `" suffix="m4a" contentType="audio/mp4" isVideo="false" path="` + xmlString(songPath) + `" type="music" transcodedSuffix="mp3" transcodedContentType="audio/mpeg"/>`
		}
	}

	content += `</directory>`
	connection.responseSuccess( content )
}
