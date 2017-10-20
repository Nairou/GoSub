package main

import (
	"database/sql"
	"os"
	"path"
	"strconv"
	"time"
	_ "github.com/mxk/go-sqlite/sqlite3"
)

func (connection *ClientConnection) getRandomSongs() {
	var content string
	size,error := strconv.Atoi( connection.parameters["size"] )
	if error != nil {
		size = 10
	}
	db,error := sql.Open( "sqlite3", settings.Database )
	if error != nil {
		println( "Error opening database", error )
		connection.responseError( 0 )
		return
	}
	defer db.Close()
	content = `<randomSongs>`

	// List some random songs
	childSongRows,error := db.Query( "SELECT songKey,songPath,songCreated,songTitle,songArtist,songAlbum,songDuration,songYear,songTrack,songDisc,songParent,folderPath,folderRoot FROM songs,folders WHERE folderKey=songParent ORDER BY RANDOM() LIMIT ?;", size )
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
		var songParent int
		var folderPath string
		var folderRoot string
		childSongRows.Scan( &songKey, &songPath, &songCreated, &songTitle, &songArtist, &songAlbum, &songDuration, &songYear, &songTrack, &songDisc, &songParent, &folderPath, &folderRoot )
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
			content += `<song id="` + strconv.Itoa(songKey) + `" parent="` + strconv.Itoa(songParent) + `" title="` + xmlString(songTitle) + `" album="` + xmlString(songAlbum) + `" artist="` + xmlString(songArtist) + `" isDir="false" coverArt="` + strconv.Itoa(songParent) + `" created="` + timestamp.Format( time.RFC3339 ) + `" duration="` + strconv.Itoa(songDuration) + `" track="` + strconv.Itoa(songTrack) + `" discNumber="` + strconv.Itoa(songDisc) + `" year="` + strconv.Itoa(songYear) + `" suffix="m4a" contentType="audio/mp4" isVideo="false" path="` + xmlString(songPath) + `" type="music" transcodedSuffix="mp3" transcodedContentType="audio/mpeg"/>`
		} else {
			content += `<song id="` + strconv.Itoa(songKey) + `" parent="` + strconv.Itoa(songParent) + `" title="` + xmlString(songTitle) + `" album="` + xmlString(songAlbum) + `" artist="` + xmlString(songArtist) + `" isDir="false" created="` + timestamp.Format( time.RFC3339 ) + `" duration="` + strconv.Itoa(songDuration) + `" track="` + strconv.Itoa(songTrack) + `" discNumber="` + strconv.Itoa(songDisc) + `" year="` + strconv.Itoa(songYear) + `" suffix="m4a" contentType="audio/mp4" isVideo="false" path="` + xmlString(songPath) + `" type="music" transcodedSuffix="mp3" transcodedContentType="audio/mpeg"/>`
		}
	}

	content += `</randomSongs>`
	connection.responseSuccess( content )
}
