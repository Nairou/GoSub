package main

import (
	"database/sql"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
	_ "github.com/mxk/go-sqlite/sqlite3"
)

type Artist struct {
	key int
	name string
	sortName string
	lastUpdated int64
}
type ArtistByName []Artist
func (artist ArtistByName) Len() int {
	return len( artist )
}
func (artist ArtistByName) Swap( i int, j int ) {
	artist[i], artist[j] = artist[j], artist[i]
}
func (artist ArtistByName) Less( i int, j int ) bool {
	irune,_ := utf8.DecodeRuneInString( artist[i].sortName )
	jrune,_ := utf8.DecodeRuneInString( artist[j].sortName )
	iLetter := unicode.IsLetter( irune )
	jLetter := unicode.IsLetter( jrune )
	if iLetter != jLetter {
		if iLetter {
			return true
		} else {
			return false
		}
	} else {
		return artist[i].sortName < artist[j].sortName
	}
}

func (connection *ClientConnection) getIndexes() {
	var content string
	db,error := sql.Open( "sqlite3", settings.Database )
	if error != nil {
		println( "Error opening database", error )
		connection.responseError( 0 )
		return
	}
	defer db.Close()
	rows,error := db.Query( "SELECT folderKey,folderName,folderLastUpdate FROM folders WHERE folderParent=0;" )
	if error != nil {
		println( "Error reading artists from database", error )
		connection.responseError( 0 )
		return
	}
	content = `<indexes lastModified="` + strconv.FormatInt( lastMusicScan, 10 ) + `">`
	var artistList []Artist
	for rows.Next() {
		var artist Artist
		rows.Scan( &artist.key, &artist.name, &artist.lastUpdated )

		artist.sortName = strings.ToUpper(artist.name)
		for _,article := range settings.IgnoredArticles {
			artist.sortName = strings.TrimPrefix( artist.sortName, strings.ToUpper(article + " ") )
		}
		for len(artist.sortName) > 0 {
			firstRune,runeSize := utf8.DecodeRuneInString( artist.sortName )
			if unicode.IsLetter(firstRune) || unicode.IsNumber(firstRune) {
				break
			}
			artist.sortName = artist.sortName[runeSize:]
		}
		artistList = append( artistList, artist )
		println( "Artist '" + artist.name + "' will be sorted as '" + artist.sortName + "'" )
	}

	sort.Sort( ArtistByName( artistList ) )

	var currentIndex rune = 0
	for _,artist := range artistList {
		firstRune,_ := utf8.DecodeRuneInString( artist.sortName )
		if !unicode.IsLetter( firstRune ) {
			firstRune = '#'
		}
		if firstRune != currentIndex {
			if currentIndex != 0 {
				content += `</index>`
			}
			currentIndex = firstRune
			content += `<index name="` + string(currentIndex) + `">`
		}

		content += `<artist name="` + xmlString(artist.name) + `" id="` + strconv.Itoa(artist.key) + `"/>`
	}
	if currentIndex != 0 {
		content += `</index>`
	}
	content += `</indexes>`
	connection.responseSuccess( content )
}
