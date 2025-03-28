package dat2img

// copy from: https://github.com/tujiaw/wechat_dat_to_image

import (
	"fmt"
)

type Format struct {
	Header []byte
	Ext    string
}

var (
	JPG     = Format{Header: []byte{0xFF, 0xD8, 0xFF}, Ext: "jpg"}
	PNG     = Format{Header: []byte{0x89, 0x50, 0x4E, 0x47}, Ext: "png"}
	GIF     = Format{Header: []byte{0x47, 0x49, 0x46, 0x38}, Ext: "gif"}
	TIFF    = Format{Header: []byte{0x49, 0x49, 0x2A, 0x00}, Ext: "tiff"}
	BMP     = Format{Header: []byte{0x42, 0x4D}, Ext: "bmp"}
	Formats = []Format{JPG, PNG, GIF, TIFF, BMP}
)

func Dat2Image(data []byte) ([]byte, string, error) {

	if len(data) < 4 {
		return nil, "", fmt.Errorf("data length is too short: %d", len(data))
	}

	findFormat := func(data []byte, header []byte) bool {
		xorBit := data[0] ^ header[0]
		for i := 0; i < len(header); i++ {
			if data[i]^header[i] != xorBit {
				return false
			}
		}
		return true
	}

	var xorBit byte
	var find bool
	var ext string
	for _, format := range Formats {
		if find = findFormat(data, format.Header); find {
			xorBit = data[0] ^ format.Header[0]
			ext = format.Ext
			break
		}
	}

	if !find {
		return nil, "", fmt.Errorf("unknown image type: %x %x", data[0], data[1])
	}

	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ xorBit
	}

	return out, ext, nil
}
