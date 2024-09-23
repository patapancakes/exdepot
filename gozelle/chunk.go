/*
	Copyright (C) 2024  Pancakes <patapancakes@pagefault.games>

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package gozelle

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
)

type Chunk struct {
	Offset uint64 `json:"offset"`
	Length uint64 `json:"length"`

	data io.Reader
}

var ErrChunkNotPrepared = errors.New("chunk not prepared")

func (c Chunk) Read(dst []byte) (int, error) {
	if c.Length == 0 {
		return 0, io.EOF
	}

	if c.data == nil {
		return 0, ErrChunkNotPrepared
	}

	n, err := c.data.Read(dst)
	if err != nil {
		return n, err
	}

	return n, nil
}

func (c *Chunk) Prepare(key []byte, src io.ReaderAt, mode Mode) error {
	// why do zero-length chunks exist?
	if c.Length == 0 {
		return nil
	}

	chunk := make([]byte, c.Length)
	_, err := src.ReadAt(chunk, int64(c.Offset))
	if err != nil {
		return fmt.Errorf("failed to read data: %s", err)
	}

	c.data = bytes.NewReader(chunk)

	// zlib buffer sizes if encrypted, not used
	//var encSize, decSize uint32
	if mode == EncryptedCompressed {
		_, err := readUint32List(c.data, 2)
		if err != nil {
			return fmt.Errorf("failed to read value: %s", err)
		}

		//encSize = v[0] // unused
		//decSize = v[1] // unused
	}

	// decrypt
	if mode == EncryptedCompressed || mode == Encrypted {
		if key == nil {
			return fmt.Errorf("missing decryption key")
		}

		data := make([]byte, c.Length)
		_, err = c.data.Read(data)
		if err != nil {
			return fmt.Errorf("failed to read data: %s", err)
		}

		// doesn't seem like this is needed
		//if len(data)%0x10 != 0 {
		//	data = append(data, make([]byte, len(data)%0x10)...)
		//}

		ci, err := aes.NewCipher(key)
		if err != nil {
			return fmt.Errorf("failed to create aes cipher: %s", err)
		}

		cipher.NewCFBDecrypter(ci, make([]byte, 0x10)).XORKeyStream(data, data)

		// doesn't seem like this is needed either
		//if mode == Encrypted {
		//	data = data[:c.Length]
		//}

		c.data = bytes.NewReader(data)
	}

	// decompress
	if mode == EncryptedCompressed || mode == Compressed {
		zr, err := zlib.NewReader(c.data)
		if err != nil {
			return fmt.Errorf("failed to create zlib reader: %s", err)
		}

		// don't close since we're storing it for later
		//defer zr.Close()

		c.data = zr
	}

	return nil
}
