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
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/schollz/progressbar/v3"
)

func main() {
	keyfile := flag.String("keyfile", "depotkeys.json", "path to depot keys file")
	manifestdir := flag.String("manifestdir", "manifests", "path to manifests directory")
	storagedir := flag.String("storagedir", "storages", "path to storages directory")
	outdir := flag.String("outdir", "extracted", "path to output directory")
	depot := flag.Int("depot", 0, "depot id to extract")
	version := flag.Int("version", 0, "depot version to extract")
	workers := flag.Int("workers", runtime.NumCPU(), "number of extraction workers")

	flag.Parse()

	fmt.Printf("exdepot by Pancakes (patapancakes@pagefault.games)\n")
	fmt.Printf("https://github.com/patapancakes/exdepot\n")
	fmt.Printf("Depot %d Version %d\n", *depot, *version)
	fmt.Printf("Using %d extraction workers\n", *workers)

	// async related
	var wg sync.WaitGroup

	var err error

	// keys
	var keys Keys

	wg.Add(1)
	go func() {
		keys, err = readKeys(*keyfile)
		if err != nil {
			log.Fatal(err)
		}

		wg.Done()
	}()

	// manifest
	var manifest Manifest

	wg.Add(1)
	go func() {
		manifest, err = readManifest(*manifestdir, *depot, *version)
		if err != nil {
			log.Fatal(err)
		}

		wg.Done()
	}()

	// index
	var index Index

	wg.Add(1)
	go func() {
		index, err = readIndex(*storagedir, *depot)
		if err != nil {
			log.Fatal(err)
		}

		wg.Done()
	}()

	wg.Wait()

	if int(manifest.DepotID) != *depot {
		log.Printf("manifest depot id %d does not match input %d", manifest.DepotID, *depot)
	}
	if int(manifest.DepotVersion) != *version {
		log.Printf("manifest depot version %d does not match input %d", manifest.DepotVersion, *version)
	}

	// extract
	data, err := os.Open(fmt.Sprintf("%s/%d.data", *storagedir, *depot))
	if err != nil {
		log.Fatalf("failed to open data file: %s", err)
	}

	defer data.Close()

	key, ok := keys[*depot]
	if !ok {
		log.Print("couldn't find key for depot")
	}

	// create directories
	for _, i := range manifest.Items {
		if !i.IsDirectory() {
			continue
		}

		err := os.MkdirAll(fmt.Sprintf("%s/%s", *outdir, i.Path), 0755)
		if err != nil {
			log.Fatalf("failed to create directory: %s", err)
		}
	}

	jobs := make(chan ExtractorJob)

	for range *workers {
		wg.Add(1)
		go extractorWorker(&wg, jobs, data, key)
	}

	bar := progressbar.Default(int64(len(manifest.Items)), "Extracting")

	// create files
	for _, i := range manifest.Items {
		bar.Add(1)

		if i.IsDirectory() {
			continue
		}

		jobs <- ExtractorJob{
			Path:  fmt.Sprintf("%s/%s", *outdir, i.Path),
			Index: index[int(i.ID)],
		}
	}

	close(jobs)

	wg.Wait()
}
