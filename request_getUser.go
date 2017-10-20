package main

import (
)

func (connection *ClientConnection) getUser() {
	var content string
	if connection.parameters["username"] != connection.parameters["u"] {
		println( "Invalid user request" )
		connection.responseError( 50 )
		return
	}

	content = `<user username="` + connection.parameters["username"] + `" scrobblingEnabled="true" adminRole="false" settingsRole="true" downloadRole="true" uploadRole="false" playlistRole="true" coverArtRole="true" commentRole="true" podcastRole="true" streamRole="true" jukeboxRole="false" shareRole="false" />`
	connection.responseSuccess( content )
}
