package state

import (
	"encoding/json" // converts structs to json
	"os"
	"path/filepath"
)

// the image is ultimately the metadata + the ordered list of layers

// struct for the image
type Image struct {
	Name    string   `json:"name"` // eg -> myapp
	Tag     string   `json:"tag"` // eg -> myapp:latest
	Digest  string   `json:"digest"` // unique ID for the image
	Created string   `json:"created"`
	Config  Config   `json:"config"` // runtime config, nested struct from the below one
	Layers  []Layer  `json:"layers"` // array of the differents layers of the img
}

type Config struct {
	Env        []string `json:"Env"` // env variables set
	Cmd        []string `json:"Cmd"` // default cmd's, eg->['python3', 'app.py']
	WorkingDir string   `json:"WorkingDir"` // the workingDir inside the container
}

// everytime a change is made to the iamge fs, a new layer is created
// these layers are immutable and can be reused across images
type Layer struct {
	Digest    string `json:"digest"` // unique hash for that layer
	Size      int64  `json:"size"` // size of the layer in bytes
	CreatedBy string `json:"createdBy"` // the cmd creating this, eg -> RUN pip install flask
}

// gets the dir in which images are stored
// eg -> /home/rahul/.docksmith/images/
func getImagesDir() string {
	return filepath.Join(getBaseDir(), "images")
}

// write an image struct to disk (the images dir)
func SaveImage(img Image) error {
	// eg -> myapp_latest.json
	path := filepath.Join(getImagesDir(), img.Name+"_"+img.Tag+".json")

	// json.MarshalIndent() converts go struct (in memory data) to json data, which is writen to disk
	data, err := json.MarshalIndent(img, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644) // 0644 -> read by all, write by owner
}

// loads
func LoadAllImages() ([]Image, error) {
	dir := getImagesDir()

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var images []Image

	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}

		var img Image
		// json.Unmarshal, takes the json data for the img and converts it to in memory go structs which our pgm can act on
		err = json.Unmarshal(data, &img)
		if err != nil {
			return nil, err
		}

		images = append(images, img)
	}

	return images, nil
}
