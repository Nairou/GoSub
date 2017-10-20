package main

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
	_ "github.com/mxk/go-sqlite/sqlite3"
)

var lastMusicScan int64

func getNextKey( transaction *sql.Tx ) int {
	// Determine the next available shared key value
	rowFolderKey := transaction.QueryRow( "SELECT max(folderKey) FROM folders;" )
	var folderMax int
	rowFolderKey.Scan( &folderMax )
	rowSongKey := transaction.QueryRow( "SELECT max(songKey) FROM songs;" )
	var songMax int
	rowSongKey.Scan( &songMax )
	nextKey := songMax + 1
	if folderMax > songMax {
		nextKey = folderMax + 1
	}
	return nextKey
}

func processSong( folderRoot string, songPath string, songParent int, quickScan bool, transaction *sql.Tx ) {
	songFile := path.Join( folderRoot, songPath )

	songFolder := path.Dir( songFile )
	coverArtExists := true
	coverArtPath := path.Join( songFolder, "cover.jpg" )
	if _,error := os.Stat( coverArtPath ); error != nil {
		coverArtPath = path.Join( songFolder, "cover.png" )
		if _,error := os.Stat( coverArtPath ); error != nil {
			coverArtExists = false
		}
	}

	fileExtension := path.Ext( songFile )
	if fileExtension == ".m4a" {
		rowSong := transaction.QueryRow( "SELECT songKey FROM songs WHERE songPath=? AND songParent=?;", songPath, songParent )
		var songKey int
		rowSong.Scan( &songKey )
		if songKey != 0 {
			if quickScan {
				_,error := transaction.Exec( "UPDATE songs SET songLastUpdate=? WHERE songKey=?;", lastMusicScan, songKey )
				if error != nil {
					println( "Error updating artist in database.", error )
				}
			} else {
				fileTag := processFileM4A( songFile )
				_,error := transaction.Exec( "UPDATE songs SET songLastUpdate=?,songTitle=?,songArtist=?,songAlbum=?,songDuration=?,songYear=?,songTrack=?,songDisc=? WHERE songKey=?;", lastMusicScan, fileTag.Title, fileTag.Artist, fileTag.Album, fileTag.Duration, fileTag.Year, fileTag.Track, fileTag.Disc, songKey )
				if error != nil {
					println( "Error updating artist in database.", error )
				}
			}
		} else {
			fileTag := processFileM4A( songFile )
			println( "    Adding song:",fileTag.Title )
			songKey = getNextKey( transaction )
			_,error := transaction.Exec( "INSERT INTO songs (songKey,songPath,songParent,songCreated,songLastUpdate,songTitle,songArtist,songAlbum,songDuration,songYear,songTrack,songDisc) VALUES (?,?,?,?,?,?,?,?,?,?,?,?);", songKey, songPath, songParent, lastMusicScan, lastMusicScan, fileTag.Title, fileTag.Artist, fileTag.Album, fileTag.Duration, fileTag.Year, fileTag.Track, fileTag.Disc )
			if error != nil {
				println( "Error inserting song into database.", error )
			}
		}
		if !coverArtExists {
			coverArtData,error := extractCoverArtM4A( songFile )
			if error == nil {
				var fileName string = "cover.jpg"
				if coverArtData.isPng {
					fileName = "cover.png"
				}
				outputFile,error := os.Create( path.Join( songFolder, fileName ) )
				if error != nil {
					println( "Error creating cover art image.", error.Error() )
				} else {
					println( "Writing cover art image in",songFolder )
					outputFile.Write( coverArtData.data )
					outputFile.Close()
				}
			}
		}
	}
}

func processFolder( folderRoot string, folderPath string, folderParent int, quickScan bool, transaction *sql.Tx ) {
	dirList,error := ioutil.ReadDir( path.Join( folderRoot, folderPath ) )
	if error != nil {
		println( "Error reading folder.", folderPath, error )
		return
	}
	for _,item := range dirList {
		if strings.HasPrefix( item.Name(), "." ) {
			continue
		}

		itemPath := path.Join( folderPath, item.Name() )
		if item.IsDir() {
			rowFolder := transaction.QueryRow( "SELECT folderKey FROM folders WHERE folderName=? AND folderParent=?;", item.Name(), folderParent )
			var folderKey int
			rowFolder.Scan( &folderKey )
			if folderKey != 0 {
				_,error := transaction.Exec( "UPDATE folders SET folderLastUpdate=? WHERE folderKey=?;", lastMusicScan, folderKey )
				if error != nil {
					println( "Error updating artist in database.", error )
				}
			} else {
				if folderParent == 0 {
					println( "Adding artist: " + item.Name() )
				} else {
					println( "  Adding folder: " + item.Name() )
				}
				folderKey = getNextKey( transaction )
				result,error := transaction.Exec( "INSERT INTO folders (folderKey,folderName,folderPath,folderParent,folderCreated,folderLastUpdate,folderRoot) VALUES (?,?,?,?,?,?,?);", folderKey, item.Name(), itemPath, folderParent, lastMusicScan, lastMusicScan, folderRoot )
				if error != nil {
					println( "Error inserting artist into database.", error )
				}
				id,error := result.LastInsertId()
				folderKey = int(id)
			}
			processFolder( folderRoot, itemPath, folderKey, quickScan, transaction )
		} else {
			processSong( folderRoot, itemPath, folderParent, quickScan, transaction )
		}
	}
}

func scanMusicFolders( quickScan bool ) {
	db,error := sql.Open( "sqlite3", settings.Database )
	if error != nil {
		println( "Error opening database", error )
		return
	}
	defer db.Close()

	artistTransaction,error := db.Begin()
	if error != nil {
		println( "Error starting database transaction.", error )
		return
	}
	lastMusicScan = time.Now().Unix()
	for _,folder := range settings.Folders {
		processFolder( folder, "", 0, quickScan, artistTransaction )
	}

	// Delete any artist records that were not updated (and therefore no longer exist on disk)
	artistResult,error := artistTransaction.Exec( "DELETE FROM folders WHERE folderLastUpdate<?;", lastMusicScan )
	if error != nil {
		println( "Error cleaning old artists from database.", error.Error() )
	}
	artistCount,error := artistResult.RowsAffected()
	if error != nil {
		println( "Error counting old artists from database.", error.Error() )
	}
	songResult,error := artistTransaction.Exec( "DELETE FROM songs WHERE songLastUpdate<?;", lastMusicScan )
	if error != nil {
		println( "Error cleaning old songs from database.", error.Error() )
	}
	songCount,error := songResult.RowsAffected()
	if error != nil {
		println( "Error counting old artists from database.", error.Error() )
	}
	println( "Deleted", artistCount, "missing artists,", songCount, "missing songs." )
	artistTransaction.Commit()
	db.Close()
}
