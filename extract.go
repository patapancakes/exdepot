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

package main

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

type ExtractorJob struct {
	Path  string
	Index IndexEntry
}

func extractorWorker(wg *sync.WaitGroup, jobs chan ExtractorJob, data io.ReaderAt, key []byte) {
	for {
		job, ok := <-jobs
		if !ok {
			wg.Done()
			return
		}

		out, err := os.OpenFile(job.Path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
		if err != nil {
			log.Fatalf("failed to open output file: %s", err)
		}

		err = extractFile(data, out, key, job.Index)
		if err != nil {
			log.Fatalf("failed to extract file: %s", err)
		}

		err = out.Sync()
		if err != nil {
			log.Fatalf("failed to sync output file: %s", err)
		}

		err = out.Close()
		if err != nil {
			log.Fatalf("failed to close output file: %s", err)
		}
	}
}

func extractFile(data io.ReaderAt, out io.Writer, key []byte, index IndexEntry) error {
	for _, cd := range index.Chunks {
		if cd.Length == 0 {
			continue
		}

		chunk := make([]byte, cd.Length)
		_, err := data.ReadAt(chunk, int64(cd.Start))
		if err != nil {
			return fmt.Errorf("failed to read data: %s", err)
		}

		var r io.Reader

		r = bytes.NewReader(chunk)

		// zlib buffer sizes if encrypted, not used
		//var encSize, decSize uint32
		if index.Mode == 2 {
			_, err := readUint32List(r, 2)
			if err != nil {
				return fmt.Errorf("failed to read value: %s", err)
			}

			//encSize = v[0] // unused
			//decSize = v[1] // unused
		}

		// decrypt
		if index.Mode == 2 || index.Mode == 3 {
			if key == nil {
				return fmt.Errorf("missing decryption key")
			}

			d := make([]byte, cd.Length)
			_, err = r.Read(d)
			if err != nil {
				return fmt.Errorf("failed to read data: %s", err)
			}

			// unneeded?
			//if len(d)%0x10 != 0 {
			//	d = append(d, make([]byte, len(d)%0x10)...)
			//}

			c, err := aes.NewCipher(key)
			if err != nil {
				return fmt.Errorf("failed to create aes cipher: %s", err)
			}

			cipher.NewCFBDecrypter(c, make([]byte, 0x10)).XORKeyStream(d, d)

			if index.Mode == 3 {
				d = d[:cd.Length]
			}

			r = bytes.NewReader(d)
		}

		// decompress
		if index.Mode == 1 || index.Mode == 2 {
			zr, err := zlib.NewReader(r)
			if err != nil {
				return fmt.Errorf("failed to create zlib reader: %s", err)
			}

			defer zr.Close()

			r = zr
		}

		_, err = io.Copy(out, r)
		if err != nil {
			return fmt.Errorf("failed to write data to output file: %s", err)
		}
	}

	return nil
}
