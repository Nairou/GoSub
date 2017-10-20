package main

import (
	"os"
	"strconv"
)

type M4AFileBlock struct {
	File *os.File
	Name string
	Length int
	Offset int64
}

func (block *M4AFileBlock) readBlockHeader() error {
	headerBuffer := make( []byte, 8 )
	_,error := block.File.ReadAt( headerBuffer, block.Offset )
	if error != nil {
		return error
	}
	block.Length = (int(headerBuffer[0]) << 24) | (int(headerBuffer[1]) << 16) | (int(headerBuffer[2]) << 8) | (int(headerBuffer[3]))
	block.Name = string(headerBuffer[4]) + string(headerBuffer[5]) + string(headerBuffer[6]) + string(headerBuffer[7])

	return nil
}

func (block *M4AFileBlock) findBlock( name string ) error {
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
			block.Offset += int64(block.Length)
		}
	}
}

func (block *M4AFileBlock) readBlockDataString() string {
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

func (block *M4AFileBlock) readBlockDataInt() int {
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

func processFileM4A( name string ) Tags {
	var block M4AFileBlock = M4AFileBlock{ nil, "", 0, 0 }
	var newTag Tags

	audioFile,error := os.Open( name )
	defer audioFile.Close()
	if error != nil {
		println( "Unable to open audio file.", name, error )
		return newTag
	}
	block.File = audioFile

	// Find song length
	block.Offset = 0
	block.findBlock( "moov" )
	block.findBlock( "trak" )
	block.findBlock( "mdia" )
	block.findBlock( "mdhd" )
	header := make( []byte, block.Length )
	_,error = block.File.ReadAt( header, block.Offset )
	if error != nil {
		println( "Unable to read audio data.", error.Error() )
		return newTag
	}
	if header[0] == 1 {
		var timeScale int64 = 0
		timeScale |= int64(header[20]) << 24
		timeScale |= int64(header[21]) << 16
		timeScale |= int64(header[22]) << 8
		timeScale |= int64(header[23])
		var duration int64 = 0
		duration |= int64(header[24]) << 56
		duration |= int64(header[25]) << 48
		duration |= int64(header[26]) << 40
		duration |= int64(header[27]) << 32
		duration |= int64(header[28]) << 24
		duration |= int64(header[29]) << 16
		duration |= int64(header[30]) << 8
		duration |= int64(header[31])
		newTag.Duration = int(duration / timeScale)
	} else {
		var timeScale int = 0
		timeScale |= int(header[12]) << 24
		timeScale |= int(header[13]) << 16
		timeScale |= int(header[14]) << 8
		timeScale |= int(header[15])
		var duration int = 0
		duration |= int(header[16]) << 24
		duration |= int(header[17]) << 16
		duration |= int(header[18]) << 8
		duration |= int(header[19])
		newTag.Duration = duration / timeScale
	}

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

type CoverArtData struct {
	data []byte
	isJpeg bool
	isPng bool
}

func extractCoverArtM4A( name string ) (CoverArtData,error) {
	var block M4AFileBlock = M4AFileBlock{ nil, "", 0, 0 }
	image := CoverArtData{ nil, false, false }

	audioFile,error := os.Open( name )
	defer audioFile.Close()
	if error != nil {
		println( "Unable to open audio file.", name, error )
		return image,error
	}
	block.File = audioFile

	block.Offset = 0
	block.findBlock( "moov" )
	block.findBlock( "udta" )
	block.findBlock( "meta" )
	block.Offset += 4
	block.findBlock( "ilst" )
	error = block.findBlock( "covr" )
	if error != nil {
		return image,error
	}
	image.data = make( []byte, block.Length )
	_,error = block.File.ReadAt( image.data, block.Offset )
	if error != nil {
		println( "Unable to read cover art.", name, error.Error() )
		return image,error
	}
	childBlock := string(image.data[4]) + string(image.data[5]) + string(image.data[6]) + string(image.data[7])
	if childBlock != "data" {
		println( "Invalid cover art data.", name, error.Error() )
		return image,error
	}
	imageType := (int(image.data[8]) << 24) | (int(image.data[9]) << 16) | (int(image.data[10]) << 8) | (int(image.data[11]))
	if imageType == 0x0D {
		image.isJpeg = true
	}
	if imageType == 0x0E {
		image.isPng = true
	}
	if !image.isJpeg && !image.isPng {
		println( "Invalid cover art image format.", name, error.Error() )
		return image,error
	}

	image.data = image.data[16:]
	return image,nil
}
