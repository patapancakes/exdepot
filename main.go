/*
	Copyright (C) 2024-2026  Pancakes <patapancakes@pagefault.games>

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
	"crypto/cipher"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/patapancakes/exdepot/gozelle"
	"github.com/patapancakes/exdepot/gozelle/gfs"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"

	_ "embed"
)

//go:embed depotkeys.json
var depotKeys []byte

func main() {
	manifestdir := flag.String("manifestdir", "manifests", "path to manifests directory")
	storagedir := flag.String("storagedir", "storages", "path to storages directory")
	outpath := flag.String("outpath", "", "path to output directory or file")
	depot := flag.Int("depot", 0, "depot id to extract")
	version := flag.Int("version", 0, "depot version to extract")
	workers := flag.Int("workers", runtime.NumCPU(), "number of extraction workers")
	mode := flag.String("mode", "extract", "mode to use (extract, validate, filelist, manifestjson, indexjson)")

	flag.Parse()

	// "interactive" mode
	if *mode == "extract" || *outpath != "" {
		fmt.Printf("exdepot by Pancakes (patapancakes@pagefault.games)\n")
		fmt.Printf("https://github.com/patapancakes/exdepot\n")
		fmt.Printf("Depot %d Version %d\n", *depot, *version)
	}

	// async related
	var eg errgroup.Group

	// keys
	var block cipher.Block
	eg.Go(func() error {
		keys, err := gozelle.ReadKeys(bytes.NewReader(depotKeys))
		if err != nil {
			return fmt.Errorf("failed to read keys file: %w", err)
		}

		block, err = keys.CipherBlockFromID(*depot)
		if err != nil && err != gozelle.ErrKeyNotFound {
			return fmt.Errorf("failed to create depot cipher block: %w", err)
		}

		return nil
	})

	// manifest
	var manifest gozelle.Manifest
	eg.Go(func() error {
		f, err := os.Open(filepath.Join(*manifestdir, fmt.Sprintf("%d_%d.manifest", *depot, *version)))
		if err != nil {
			return fmt.Errorf("failed to open manifest file: %w", err)
		}

		defer f.Close()

		manifest, err = gozelle.ReadManifest(f)
		if err != nil {
			return err
		}

		return nil
	})

	// index
	var index gozelle.Index
	eg.Go(func() error {
		f, err := os.Open(filepath.Join(*storagedir, fmt.Sprintf("%d.index", *depot)))
		if err != nil {
			return fmt.Errorf("failed to open index file: %w", err)
		}

		defer f.Close()

		index, err = gozelle.ReadIndex(f)
		if err != nil {
			return err
		}

		return nil
	})

	err := eg.Wait()
	if err != nil {
		log.Fatal(err)
	}

	switch *mode {
	case "extract":
		err = doExtract(*storagedir, *outpath, *workers, block, manifest, index)
	case "web":
		err = doWeb(*storagedir, block, manifest, index)
	case "validate":
		err = fmt.Errorf("not implemented yet")
	case "filelist":
		err = doFileList(manifest, *outpath)
	case "manifestjson":
		err = doManifestJSON(manifest, *outpath)
	case "indexjson":
		err = doIndexJSON(index, *outpath)
	default:
		err = fmt.Errorf("unknown mode %s", *mode)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func doExtract(storagedir string, outpath string, workers int, block cipher.Block, manifest gozelle.Manifest, index gozelle.Index) error {
	fmt.Printf("Using %d extraction workers\n", workers)

	if outpath == "" {
		outpath = fmt.Sprintf("%d_%d", manifest.DepotID, manifest.DepotVersion)
	}

	// extract
	data, err := os.Open(filepath.Join(storagedir, fmt.Sprintf("%d.data", manifest.DepotID)))
	if err != nil {
		return fmt.Errorf("failed to open data file: %s", err)
	}

	defer data.Close()

	var eg errgroup.Group
	eg.SetLimit(workers)

	bar := progressbar.Default(int64(len(manifest.Items)), "Extracting")

	// create directories and files
	err = os.MkdirAll(outpath, 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create output directory: %s", err)
	}

	for _, i := range manifest.Items {
		bar.Add(1)

		if i.IsDirectory() {
			err := os.Mkdir(filepath.Join(outpath, i.Path), 0755)
			if err != nil && !os.IsExist(err) {
				return fmt.Errorf("failed to create directory: %s", err)
			}

			continue
		}

		eg.Go(func() error {
			return extractorWorker(filepath.Join(outpath, i.Path), index[uint64(i.ID)], data, block)
		})
	}

	return eg.Wait()
}

func doWeb(storagedir string, block cipher.Block, manifest gozelle.Manifest, index gozelle.Index) error {
	data, err := os.Open(filepath.Join(storagedir, fmt.Sprintf("%d.data", manifest.DepotID)))
	if err != nil {
		return fmt.Errorf("failed to open data file: %s", err)
	}

	defer data.Close()

	fs, _ := gfs.NewFS(manifest, index, block, data)

	http.Handle("/", http.FileServerFS(fs))
	return http.ListenAndServe(":8000", nil)
}

func doFileList(manifest gozelle.Manifest, outpath string) error {
	w := os.Stdout
	if outpath != "" {
		var err error
		w, err = os.OpenFile(outpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open output file: %s", err)
		}
	}

	for _, i := range manifest.Items {
		if i.Path == "" {
			continue
		}

		_, err := w.Write([]byte(i.Path + "\n"))
		if err != nil {
			return fmt.Errorf("failed to write to output file: %s", err)
		}
	}

	return nil
}

func doManifestJSON(manifest gozelle.Manifest, outpath string) error {
	w := os.Stdout
	if outpath != "" {
		var err error
		w, err = os.OpenFile(outpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open output file: %s", err)
		}
	}

	err := json.NewEncoder(w).Encode(manifest)
	if err != nil {
		return fmt.Errorf("failed to encode output json: %s", err)
	}

	return nil
}

func doIndexJSON(index gozelle.Index, outpath string) error {
	w := os.Stdout
	if outpath != "" {
		var err error
		w, err = os.OpenFile(outpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open output file: %s", err)
		}
	}

	err := json.NewEncoder(w).Encode(index)
	if err != nil {
		return fmt.Errorf("failed to encode output json: %s", err)
	}

	return nil
}
