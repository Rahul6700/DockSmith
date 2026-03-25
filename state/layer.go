package state

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// takes in folder path
// outputs a layer (a tar bsically)
// the folder -> tar -> hash -> stored

// example take the dir docksmith
// we take all the file paths state/hash.go, state/layer.go, etc
// create a tar of all this and then content hash it
// store it -> ~/.docksmith/layers/sha256:abc123.tar
// the function return sumn like ->
//Layer{
//  Digest: "sha256:abc123",
//  Size: 2048
//}
func CreateLayerFromDir(srcDir string) (Layer, error) {

	// creating a temp .tar file
	tmpFile, err := os.CreateTemp("", "layer-*.tar")
	if err != nil {
		return Layer{}, err
	}
	defer tmpFile.Close()

	// tw is a tool that writes in the tar format
	tw := tar.NewWriter(tmpFile)
	defer tw.Close()

	// files arr is to store all the file paths of the func param dir
	var files []string

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return Layer{}, err
	}

	// we sort the file paths (this helps us make sure that same tar -> same hash)
	sort.Strings(files)

	// go through all the files in the arr
	for _, path := range files {

		info, err := os.Stat(path) // get files dets, like file size, permissions, etc
		if err != nil {
			return Layer{}, err
		}

		relPath, err := filepath.Rel(srcDir, path) // getting relative path -> state/layer.go to layer.go
		if err != nil {
			return Layer{}, err
		}

		header, err := tar.FileInfoHeader(info, "") // header is the metadata for tarentry
		if err != nil {
			return Layer{}, err
		}
		header.Name = relPath // setting the file name as the relative name

		// setting all the time stamp related values to 0
		// we are not tracking timestamp
		// if we do then, same tar's with different time stamp will have diferent hashes
		// we want to reuse tar's (layers), so we need them to have the same hash, hence no timestamp
		header.ModTime = header.ModTime.Add(-header.ModTime.Sub(header.ModTime))
		header.AccessTime = header.AccessTime.Add(-header.AccessTime.Sub(header.AccessTime))
		header.ChangeTime = header.ChangeTime.Add(-header.ChangeTime.Sub(header.ChangeTime))

		err = tw.WriteHeader(header) // add the file metadata to tar
		if err != nil {
			return Layer{}, err
		}

		// open the files, write the contents to the tar
		file, err := os.Open(path)
		if err != nil {
			return Layer{}, err
		}

		_, err = io.Copy(tw, file)
		file.Close()
		if err != nil {
			return Layer{}, err
		}
	}

	// now we have the tar file ready in our tmpFile
	err = tw.Close()
	if err != nil {
		return Layer{}, err
	}

	err = tmpFile.Close()
	if err != nil {
		return Layer{}, err
	}

	data, err := os.ReadFile(tmpFile.Name()) // read the content of the tmpFile
	if err != nil {
		return Layer{}, err
	}

	digest := HashBytes(data) // creating the hash for the tar (using content hashing)

	// creating a final filepath in the format of -> ~/.docksmith/layers/sha256:abc123.tar
	finalPath := filepath.Join(getBaseDir(), "layers", digest+".tar")

	// if the file does not exist, write it to disk
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		fmt.Println("Writing new layer to disk...")
		err = os.WriteFile(finalPath, data, 0644)
		if err != nil {
			return Layer{}, err
		}
	} else {
	fmt.Println("Layer already exists, skipping write")
	}

	// returning the layer
	return Layer{
		Digest: digest,
		Size:   int64(len(data)),
	}, nil

}
