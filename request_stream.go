package main

import (
	"os/exec"
	"database/sql"
	"path"
	"strconv"
	_ "github.com/mxk/go-sqlite/sqlite3"
)

func (connection *ClientConnection) stream() {
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
	songRow := db.QueryRow( "SELECT songPath,folderRoot FROM songs,folders WHERE songKey=? AND folderKey=songParent;", id )
	var songPath string
	var folderRoot string
	error = songRow.Scan( &songPath, &folderRoot )
	if error != nil {
		println( "Error reading song path from database", error.Error() )
		connection.responseError( 70 )
		return
	}
	songPath = path.Join( folderRoot, songPath )
	println( "Song path", songPath )
	command := exec.Command( "ffmpeg", "-i", songPath, "-vn", "-ab", "128k", "-v", "0", "-f", "mp3", "-" )
	//command.Stderr = os.Stderr
	command.Stdout = connection.output
	command.Run()
}
