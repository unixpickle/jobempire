package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"os"
	"time"
	"io/ioutil"
	"path"
	"path/filepath"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindata_file_info struct {
	name string
	size int64
	mode os.FileMode
	modTime time.Time
}

func (fi bindata_file_info) Name() string {
	return fi.name
}
func (fi bindata_file_info) Size() int64 {
	return fi.size
}
func (fi bindata_file_info) Mode() os.FileMode {
	return fi.mode
}
func (fi bindata_file_info) ModTime() time.Time {
	return fi.modTime
}
func (fi bindata_file_info) IsDir() bool {
	return false
}
func (fi bindata_file_info) Sys() interface{} {
	return nil
}

var _assets_header_html = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xb2\xc9\x4b\x2c\x53\xc8\x4c\xb1\x55\xca\x48\x4d\x4c\x49\x2d\x52\xb2\xe3\x52\x50\xa8\xae\xce\x4c\x53\x48\x2d\x54\xd0\x53\x50\xca\xca\x4f\x2a\x56\xaa\xad\x05\x0a\xda\x24\x2a\x24\xe7\x24\x16\x17\xc3\x54\xea\x66\x96\xa4\xe6\x2a\x20\xb1\x75\x93\x4b\x8b\x94\x14\x32\x8a\x52\xd3\x6c\x95\xf4\xc1\xfa\xec\xbc\x80\xa4\x8d\x7e\x22\xc4\xcc\xd4\x9c\xe2\x54\xdc\x26\xe1\xd5\x99\x97\x02\xd4\x88\xea\xb0\xe2\x9c\xc4\xb2\x54\xb2\x9c\x06\xd5\x69\x17\x0c\xa6\x49\x75\x1e\x2e\xdd\x60\x27\xda\xe8\x03\x43\xd3\x8e\x0b\x10\x00\x00\xff\xff\x82\xd0\xc9\x50\x54\x01\x00\x00")

func assets_header_html_bytes() ([]byte, error) {
	return bindata_read(
		_assets_header_html,
		"assets/header.html",
	)
}

func assets_header_html() (*asset, error) {
	bytes, err := assets_header_html_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "assets/header.html", size: 340, mode: os.FileMode(420), modTime: time.Unix(1475100088, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _assets_jobs_html = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x4c\x50\xcb\x6e\xc3\x30\x0c\xbb\xf7\x2b\x34\x9d\x36\xa0\x6d\xb6\xdb\x0e\x71\x3f\x60\x7f\xe1\x38\x2c\xec\xcd\x8f\xc2\x56\xda\x06\x45\xff\x7d\x8e\x83\x00\x3d\x49\x24\x45\x50\x60\xff\x36\x26\x23\xf3\x05\x64\x25\xf8\xd3\xae\x5f\x07\x51\x6f\xa1\xc7\x65\xa9\x6b\x80\x68\x32\x56\xe7\x02\x51\x3c\xc9\xf9\xf0\xcd\xaf\x52\xd4\x01\x8a\xaf\x0e\xb7\x4b\xca\xc2\x4d\x21\x32\x29\x0a\x62\x35\xdc\xdc\x28\x56\x8d\xb8\x3a\x83\x43\x03\x7b\x72\xd1\x89\xd3\xfe\x50\x8c\xf6\x50\x5f\xc7\xcf\x3d\x05\x7d\x77\x61\x0a\xaf\xd4\x54\x90\x1b\xd6\x43\xa5\x62\xda\x52\xc5\x89\xc7\xe9\x27\x0d\x85\xde\x7f\xd3\x80\x70\x71\x19\x1f\x7d\xb7\xf2\xbb\xf5\xc8\xbb\xf8\x47\x19\x5e\x71\x91\xd9\xa3\x58\x40\x98\x6c\xc6\x79\x63\xba\x36\x8e\xa6\x14\xa6\xa5\x03\xc5\x82\xbb\x74\x0b\x6e\x15\x74\x5b\x07\xfd\x90\xc6\x79\xcd\x7e\x3c\xa4\xc6\x79\x2d\x20\x5e\x64\x64\x26\xae\x3f\x14\x7e\x3e\x9b\x67\x3d\xad\xde\x56\xe4\x7f\x00\x00\x00\xff\xff\x1e\x0a\x88\x7d\x60\x01\x00\x00")

func assets_jobs_html_bytes() ([]byte, error) {
	return bindata_read(
		_assets_jobs_html,
		"assets/jobs.html",
	)
}

func assets_jobs_html() (*asset, error) {
	bytes, err := assets_jobs_html_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "assets/jobs.html", size: 352, mode: os.FileMode(420), modTime: time.Unix(1475100310, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _assets_styles_style_css = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xca\x28\xc9\xcd\xd1\x51\x48\xca\x4f\xa9\x54\xa8\xe6\x52\x50\xc8\x4d\x2c\x4a\xcf\xcc\xb3\x52\x30\xb0\x06\x72\x92\x12\x93\xb3\xd3\x8b\xf2\x4b\xf3\x52\x74\x93\xf3\x73\xf2\x8b\xac\x14\x94\x93\x4d\x53\x8d\x52\x2d\xac\xb9\x6a\xb9\xb8\x94\x33\x52\x13\x53\x52\x8b\xc0\xda\xca\x33\x53\x4a\x32\xac\x14\x0c\x0d\x0c\x54\x41\x1a\x33\x52\x33\xd3\x33\x4a\xac\x14\x8c\x0d\x0a\x2a\xb0\x1b\x54\x9e\x91\x59\x92\x0a\x32\x06\x10\x00\x00\xff\xff\x28\x6d\x67\x79\x80\x00\x00\x00")

func assets_styles_style_css_bytes() ([]byte, error) {
	return bindata_read(
		_assets_styles_style_css,
		"assets/styles/style.css",
	)
}

func assets_styles_style_css() (*asset, error) {
	bytes, err := assets_styles_style_css_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "assets/styles/style.css", size: 128, mode: os.FileMode(420), modTime: time.Unix(1475100362, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if (err != nil) {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"assets/header.html": assets_header_html,
	"assets/jobs.html": assets_jobs_html,
	"assets/styles/style.css": assets_styles_style_css,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func func() (*asset, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"assets": &_bintree_t{nil, map[string]*_bintree_t{
		"header.html": &_bintree_t{assets_header_html, map[string]*_bintree_t{
		}},
		"jobs.html": &_bintree_t{assets_jobs_html, map[string]*_bintree_t{
		}},
		"styles": &_bintree_t{nil, map[string]*_bintree_t{
			"style.css": &_bintree_t{assets_styles_style_css, map[string]*_bintree_t{
			}},
		}},
	}},
}}

// Restore an asset under the given directory
func RestoreAsset(dir, name string) error {
        data, err := Asset(name)
        if err != nil {
                return err
        }
        info, err := AssetInfo(name)
        if err != nil {
                return err
        }
        err = os.MkdirAll(_filePath(dir, path.Dir(name)), os.FileMode(0755))
        if err != nil {
                return err
        }
        err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
        if err != nil {
                return err
        }
        err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
        if err != nil {
                return err
        }
        return nil
}

// Restore assets under the given directory recursively
func RestoreAssets(dir, name string) error {
        children, err := AssetDir(name)
        if err != nil { // File
                return RestoreAsset(dir, name)
        } else { // Dir
                for _, child := range children {
                        err = RestoreAssets(dir, path.Join(name, child))
                        if err != nil {
                                return err
                        }
                }
        }
        return nil
}

func _filePath(dir, name string) string {
        cannonicalName := strings.Replace(name, "\\", "/", -1)
        return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

