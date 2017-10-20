package main

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	_ "github.com/mxk/go-sqlite/sqlite3"
)

type ClientConnection struct {
	output http.ResponseWriter
	parameters map[string]string
}

type UserLogin struct {
	Username string
	Password string
}
type Configuration struct {
	ListenPort int
	Database string
	IgnoredArticles []string
	Users []UserLogin
	Folders []string
}

var subsonicVersion string = "1.2.0"
var responseHeaderOk string = `<?xml version="1.0" encoding="UTF-8"?><subsonic-response xmlns="http://subsonic.org/restapi" status="ok" version="` + subsonicVersion + `">`
var responseHeaderFail string = `<?xml version="1.0" encoding="UTF-8"?><subsonic-response xmlns="http://subsonic.org/restapi" status="failed" version="` + subsonicVersion + `">`
var responseFooter string = `</subsonic-response>`
var settings Configuration

func (connection *ClientConnection) responseSuccess( content string ) {
	io.WriteString( connection.output, responseHeaderOk + content + responseFooter )
}
func (connection *ClientConnection) responseError( code int ) {
	var message string
	switch code {
	case 0:
		message = "Generic error."
	case 10:
		message = "Required parameter is missing."
	case 20:
		message = "Incompatible Subsonic REST protocol version. Client must upgrade."
	case 30:
		message = "Incompatible Subsonic REST protocol version. Server must upgrade."
	case 40:
		message = "Wrong username or password"
	case 50:
		message = "User is not authorized for the given operation."
	case 70:
		message = "The requested data was not found."
	}

	io.WriteString( connection.output, responseHeaderFail + `<error code="` + strconv.Itoa(code) + `" message="` + message + `"/>` + responseFooter )
}

func xmlString( text string ) string {
	output := text
	output = strings.Replace( output, "&", "&amp;", -1 )
	output = strings.Replace( output, "<", "&lt;", -1 )
	output = strings.Replace( output, ">", "&gt;", -1 )
	output = strings.Replace( output, "\"", "&quot;", -1 )
	return output
}

func (connection *ClientConnection) processRequest( request string ) {
	clientUsername := connection.parameters["u"]
	clientPassword := connection.parameters["p"]
	if strings.HasPrefix( clientPassword, "enc:" ) {
		password,error := hex.DecodeString( clientPassword[4:] )
		if error != nil {
			println( "Error decoding password" )
			connection.responseError( 40 )
			return
		}
		clientPassword = string(password)
	}
	valid := false
	for _,login := range settings.Users {
		if login.Username == clientUsername && login.Password == clientPassword {
			valid = true
			break
		}
	}
	if !valid {
		println( "Invalid login" )
		connection.responseError( 40 )
		return
	}

	println( "API:", request )
	for i,p := range connection.parameters {
		if i != "u" && i != "p" && i != "v" && i != "c" {
			println( " ", i, "=", p )
		}
	}

	switch request {
	case "/rest/ping.view":
		connection.responseSuccess( "" )
	case "/rest/getCoverArt.view":
		connection.getCoverArt()
	case "/rest/getIndexes.view":
		connection.getIndexes()
	case "/rest/getLicense.view":
		connection.responseSuccess( `<license valid="true"/>` )
	case "/rest/getMusicDirectory.view":
		connection.getMusicDirectory()
	case "/rest/getMusicFolders.view":
		connection.getMusicFolders()
	case "/rest/getRandomSongs.view":
		connection.getRandomSongs()
	case "/rest/getUser.view":
		connection.getUser()
	case "/rest/stream.view":
		connection.stream()
	default:
		connection.responseError( 0 )
	}
}

func subsonicServer( output http.ResponseWriter, request *http.Request ) {
	var connection ClientConnection
	connection.output = output
	connection.parameters = make( map[string]string )
	parameters := strings.Split( request.URL.RawQuery, "&" )
	for _, item := range parameters {
		if len(item) == 0 {
			continue
		}
		items := strings.Split( item, "=" )
		connection.parameters[items[0]] = items[1]
	}
	bodyBuffer := new( bytes.Buffer )
	bodyBuffer.ReadFrom( request.Body )
	parameters = strings.Split( bodyBuffer.String(), "&" )
	for _, item := range parameters {
		if len(item) == 0 {
			continue
		}
		items := strings.Split( item, "=" )
		connection.parameters[items[0]] = items[1]
	}
	request.Body.Close()
	connection.processRequest( request.URL.Path )
}

func main() {
	println( "Loading settings." )
	jsonFile,error := os.Open( "settings.conf" )
	defer jsonFile.Close()
	if error != nil {
		println( "Unable to read settings.", error.Error() )
		return
	}
	jsonDecoder := json.NewDecoder( jsonFile )
	error = jsonDecoder.Decode( &settings )
	if error != nil {
		println( "Unable to parse settings.", error.Error() )
		return
	}

	println( "Checking database integrity." )
	if _,error := os.Stat( settings.Database ); os.IsNotExist( error ) {
		file,_ := os.Create( settings.Database )
		file.Close()
	}
	db,error := sql.Open( "sqlite3", settings.Database )
	if error != nil {
		println( "Error opening database", error.Error() )
		return
	}
	error = db.Ping()
	if error != nil {
		println( "Error accessing database", error.Error() )
		return
	}

	_,error = db.Exec( `CREATE TABLE IF NOT EXISTS folders (
				folderKey INTEGER PRIMARY KEY,
				folderName TEXT NOT NULL,
				folderPath TEXT NOT NULL,
				folderParent INTEGER NOT NULL,
				folderCreated INTEGER NOT NULL,
				folderLastUpdate INTEGER NOT NULL,
				folderRoot TEXT NOT NULL
			);` )
	if error != nil {
		println( "Error creating 'folders' database table", error )
		return
	}
	_,error = db.Exec( `CREATE TABLE IF NOT EXISTS songs (
				songKey INTEGER PRIMARY KEY,
				songPath TEXT NOT NULL,
				songParent INTEGER NOT NULL,
				songCreated INGETER NOT NULL,
				songLastUpdate INTEGER NOT NULL,
				songTitle TEXT NOT NULL,
				songArtist TEXT NOT NULL,
				songAlbum TEXT NOT NULL,
				songDuration INTEGER NOT NULL,
				songYear INTEGER NOT NULL,
				songTrack INTEGER NOT NULL,
				songDisc INTEGER NOT NULL
			);` )
	if error != nil {
		println( "Error creating 'songs' database table", error )
		return
	}
	db.Close()

	go func() {
		counter := 0
		for {
			println( "Enumerating music." )
			scanMusicFolders( counter % 6 != 0 )
			time.Sleep( 10 * time.Minute )
			counter++
		}
	}()

	println( "Starting server." )
	http.HandleFunc( "/rest/", subsonicServer )
	error = http.ListenAndServe( ":" + strconv.Itoa(settings.ListenPort), nil )
	if error != nil {
		println( "Fatal error, unable to start server.", error )
	}
}
