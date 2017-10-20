package main

import (
	"strconv"
)

func (connection *ClientConnection) getMusicFolders() {
	content := `<musicFolders>`
	for index,folder := range settings.Folders {
		content += `<musicFolder id="` + strconv.Itoa(index) + `" name="` + xmlString(folder) + `"/>`
	}
	content += `</musicFolders>`
	connection.responseSuccess( content )
}
