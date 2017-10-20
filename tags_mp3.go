package main

import (
	"os"
	"strconv"
)

type MP3FileBlock struct {
	File *os.File
	Name string
	Length int
	Offset int64
}

func (block *MP3FileBlock) readBlockHeader() error {
	headerBuffer := make( []byte, 8 )
	_,error := block.File.ReadAt( headerBuffer, block.Offset )
	if error != nil {
		return error
	}
	block.Name = string(headerBuffer[0]) + string(headerBuffer[1]) + string(headerBuffer[2]) + string(headerBuffer[3])
	block.Length = (int(headerBuffer[4]) << 24) | (int(headerBuffer[5]) << 16) | (int(headerBuffer[6]) << 8) | (int(headerBuffer[7]))

	return nil
}

func (block *MP3FileBlock) findBlock( name string ) error {
	for {
		error := block.readBlockHeader()
		if error != nil {
			return error
		}

		if block.Name == name {
			block.Offset += 8
			block.Length -= 8
			return nil
		} else {
			block.Offset += int64(block.Length) + 10
		}
	}
}

func (block *MP3FileBlock) readBlockDataString() string {
	error := block.readBlockHeader()
	if error != nil {
		return ""
	}
	if block.Name != "data" {
		return ""
	}
	buffer := make( []byte, block.Length - 16 )
	_,error = block.File.ReadAt( buffer, block.Offset + 16 )
	if error != nil {
		return ""
	}

	return string(buffer)
}

func (block *MP3FileBlock) readBlockDataInt() int {
	error := block.readBlockHeader()
	if error != nil {
		return 0
	}
	if block.Name != "data" {
		return 0
	}
	buffer := make( []byte, block.Length - 16 )
	_,error = block.File.ReadAt( buffer, block.Offset + 16 )
	if error != nil {
		return 0
	}

	return (int(buffer[0]) << 24) | (int(buffer[1]) << 16) | (int(buffer[2]) << 8) | (int(buffer[3]))
}

func processFileMP3( name string ) Tags {
	var block MP3FileBlock = MP3FileBlock{ nil, "", 0, 0 }
	var newTag Tags

	audioFile,error := os.Open( name )
	defer audioFile.Close()
	if error != nil {
		println( "Unable to open audio file.", name, error )
		return newTag
	}
	block.File = audioFile

	// Find tag data
	block.Offset = 0
	block.findBlock( "moov" )
	block.findBlock( "udta" )
	block.findBlock( "meta" )
	block.Offset += 4
	block.findBlock( "ilst" )
	blockStart := block.Offset
	block.findBlock( "©nam" )
	newTag.Title = block.readBlockDataString()
	block.Offset = blockStart
	block.findBlock( "©alb" )
	newTag.Album = block.readBlockDataString()
	block.Offset = blockStart
	block.findBlock( "©ART" )
	newTag.Artist = block.readBlockDataString()
	block.Offset = blockStart
	block.findBlock( "aART" )
	newTag.AlbumArtist = block.readBlockDataString()
	block.Offset = blockStart
	block.findBlock( "©day" )
	yearString := block.readBlockDataString()
	if len(yearString) >= 4 {
		year,_ := strconv.Atoi( yearString[:4] )
		newTag.Year = year
	}
	block.Offset = blockStart
	block.findBlock( "trkn" )
	newTag.Track = block.readBlockDataInt()
	block.Offset = blockStart
	block.findBlock( "disk" )
	newTag.Disc = block.readBlockDataInt()

	return newTag
}
