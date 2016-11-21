package main

import (
	"archive/zip"
	"encoding/json"
        "flag"
	"fmt"
	"html/template"
        "io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beevik/etree"
)

// Metadata metadata struct
type Metadata struct {
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	Identifier string    `json:"identifier"`
	Language   string    `json:"language"`
	Modified   time.Time `json:"modified"`
}

// Link link struct
type Link struct {
	Rel      string `json:"rel,omitempty"`
	Href     string `json:"href"`
	TypeLink string `json:"type"`
	Height   int    `json:"height,omitempty"`
	Width    int    `json:"width,omitempty"`
}

// Manifest manifest struct
type Manifest struct {
	Metadata  Metadata `json:"metadata"`
	Links     []Link   `json:"links"`
	Spine     []Link   `json:"spine,omitempty"`
	Resources []Link   `json:"resources"`
}

// Icon icon struct for AppInstall
type Icon struct {
	Src       string `json:"src"`
	Size      string `json:"size"`
	MediaType string `json:"type"`
}

// AppInstall struct for app install banner
type AppInstall struct {
	ShortName string `json:"short_name"`
	Name      string `json:"name"`
	StartURL  string `json:"start_url"`
	Display   string `json:"display"`
	Icons     Icon   `json:"icons"`
}

func main() {
     var epubDir = *flag.String("epubDir", "books", "Directory of epub files to parse")
     if !strings.HasSuffix(epubDir, "/") {
       epubDir = epubDir + "/"
     }
     var outputDir = *flag.String("outputDir", "out", "Directory to put generated files in")
     if !strings.HasSuffix(outputDir, "/") {
       outputDir = outputDir + "/"
     }
     var domain = *flag.String("domain", "", "Domain where files will be hosted")
     flag.Parse()
     var books = getBooks(epubDir)
     for i := 0; i < len(books); i++ {
       var book = books[i]
       processBook(book, epubDir, outputDir, domain)
     }
}

func processBook(book string, epubDir string, outputDir string, domain string) {
    var bookOutputDir = outputDir + "/" + book
    _ = os.Mkdir(outputDir, os.ModePerm)
    _ = os.Mkdir(bookOutputDir, os.ModePerm)
    getManifest(book, domain, epubDir, outputDir)
    getWebAppManifest(book, epubDir, outputDir)
    bookIndex(book, outputDir)
    getAssets(book, epubDir, outputDir)
}

func getManifest(filename string, domain string, epubDir string, outputDir string) {
	var opfFileName string
	var manifestStruct Manifest
	var metaStruct Metadata

	metaStruct.Modified = time.Now()

	filename_path := epubDir + filename

	self := Link{
		Rel:      "self",
		Href:     domain + "/" + filename + "/manifest.json",
		TypeLink: "application/json",
	}
	manifestStruct.Links = make([]Link, 1)
	manifestStruct.Resources = make([]Link, 0)
	manifestStruct.Resources = make([]Link, 0)
	manifestStruct.Links[0] = self

	zipReader, err := zip.OpenReader(filename_path)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, f := range zipReader.File {
		if f.Name == "META-INF/container.xml" {
			rc, errOpen := f.Open()
			if errOpen != nil {
				fmt.Println("error openging " + f.Name)
			}
			doc := etree.NewDocument()
			_, err = doc.ReadFrom(rc)
			if err == nil {
				root := doc.SelectElement("container")
				rootFiles := root.SelectElements("rootfiles")
				for _, rootFileTag := range rootFiles {
					rootFile := rootFileTag.SelectElement("rootfile")
					if rootFile != nil {
						opfFileName = rootFile.SelectAttrValue("full-path", "")
					}
				}
			} else {
				fmt.Println(err)
			}
			rc.Close()
		}
	}

        var opfParts = strings.Split(opfFileName, "/")
        var opfDir = ""
        if len(opfParts) > 1 {
          opfDir = opfParts[0]
        }
	if opfFileName != "" {
		for _, f := range zipReader.File {
			if f.Name == opfFileName {
				rc, errOpen := f.Open()
				if errOpen != nil {
					fmt.Println("error openging " + f.Name)
				}
				doc := etree.NewDocument()
				_, err = doc.ReadFrom(rc)
				if err == nil {
					root := doc.SelectElement("package")
					meta := root.SelectElement("metadata")

					titleTag := meta.SelectElement("title")
					metaStruct.Title = titleTag.Text()

					langTag := meta.SelectElement("language")
					metaStruct.Language = langTag.Text()

					identifierTag := meta.SelectElement("identifier")
					metaStruct.Identifier = identifierTag.Text()

					creatorTag := meta.SelectElement("creator")
					metaStruct.Author = creatorTag.Text()

					bookManifest := root.SelectElement("manifest")
					itemsManifest := bookManifest.SelectElements("item")

                                        cacheManifestString := "CACHE MANIFEST\n# timestamp "
                                        cacheManifestString += time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006")
                                        cacheManifestString += "\n\n"

					for _, item := range itemsManifest {
						linkItem := Link{}
						linkItem.TypeLink = item.SelectAttrValue("media-type", "")
						linkItem.Href = opfDir + "/" + item.SelectAttrValue("href", "")
						if linkItem.TypeLink == "application/xhtml+xml" {
							manifestStruct.Spine = append(manifestStruct.Spine, linkItem)
						} else {
							manifestStruct.Resources = append(manifestStruct.Resources, linkItem)
						}
                                                cacheManifestString += linkItem.Href + "\n"
					}

                                        cacheManifestString += "\nNETWORK:\n*\n"

					manifestStruct.Metadata = metaStruct
					j, _ := json.Marshal(manifestStruct)
                                        ioutil.WriteFile(outputDir + filename + "/" + "manifest.json", j, 0644)
                                        ioutil.WriteFile(outputDir + filename + "/" + "manifest.appcache", []byte(cacheManifestString), 0644)
					return
				}
			}
		}
	}

}

func getWebAppManifest(filename string, epubDir string, outputDir string) {
	var opfFileName string
	var webapp AppInstall

	webapp.Display = "standalone"
	webapp.StartURL = "index.html"
	webapp.Icons = Icon{
		Size:      "144x144",
		Src:       "/logo.png",
		MediaType: "image/png",
	}

	zipReader, err := zip.OpenReader(epubDir + filename)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, f := range zipReader.File {
		if f.Name == "META-INF/container.xml" {
			rc, errOpen := f.Open()
			if errOpen != nil {
				fmt.Println("error openging " + f.Name)
			}
			doc := etree.NewDocument()
			_, err = doc.ReadFrom(rc)
			if err == nil {
				root := doc.SelectElement("container")
				rootFiles := root.SelectElements("rootfiles")
				for _, rootFileTag := range rootFiles {
					rootFile := rootFileTag.SelectElement("rootfile")
					if rootFile != nil {
						opfFileName = rootFile.SelectAttrValue("full-path", "")
					}
				}
			} else {
				fmt.Println(err)
			}
			rc.Close()
		}
	}

	if opfFileName != "" {
		for _, f := range zipReader.File {
			if f.Name == opfFileName {
				rc, errOpen := f.Open()
				if errOpen != nil {
					fmt.Println("error openging " + f.Name)
				}
				doc := etree.NewDocument()
				_, err = doc.ReadFrom(rc)
				if err == nil {
					root := doc.SelectElement("package")
					meta := root.SelectElement("metadata")

					titleTag := meta.SelectElement("title")
					webapp.Name = titleTag.Text()
					webapp.ShortName = titleTag.Text()

					j, _ := json.Marshal(webapp)
                                        ioutil.WriteFile(outputDir + filename + "/" + "webapp.webmanifest", j, 0644)
					return
				}
			}
		}
	}

}

func bookIndex(book string, outputDir string) {
	var err error

	filename := outputDir + book

	t, err := template.ParseFiles("index.html") // Parse template file.
	if err != nil {
		fmt.Println(err)
	}
        f, err := os.Create(filename + "/index.html")
        if err != nil {
           fmt.Println("create file: ", err)
        }
	t.Execute(f, filename)
        f.Close()
}

func getBooks(epubDir string) []string {
	var books []string

	files, _ := ioutil.ReadDir(epubDir)
	for _, f := range files {
		fmt.Println(f.Name())
		books = append(books, f.Name())
	}

        return books
}

func getAssets(filename string, epubDir string, outputDir string) {
	filename_path := epubDir + filename

	zipReader, err := zip.OpenReader(filename_path)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, f := range zipReader.File {
                rc, errOpen := f.Open()
		if errOpen != nil {
			fmt.Println("error openging " + f.Name)
		}
                defer rc.Close()
                fpath := filepath.Join(outputDir + filename, f.Name)

                if f.FileInfo().IsDir() {
                   os.MkdirAll(fpath, os.ModePerm)
                } else {
                   var fdir string
                   if lastIndex := strings.LastIndex(fpath,string(os.PathSeparator)); lastIndex > -1 {
                       fdir = fpath[:lastIndex]
                   }

                   err = os.MkdirAll(fdir, os.ModePerm)
                   if err != nil {
                       fmt.Println("err ", err)
                       return
                   }
                   f, err := os.OpenFile(
                       fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
                   if err != nil {
                       fmt.Println("err ", err)
                       return
                   }
                   defer f.Close()

                   _, err = io.Copy(f, rc)
                   if err != nil {
                       fmt.Println("err ", err)
                       return
                   }
                }
        }
}