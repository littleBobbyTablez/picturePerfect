package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// Image represents a single image file
type Image struct {
	Name string
	Path string
	URL  string
}

// GalleryData holds the data for our template
type GalleryData struct {
	Images []Image
	Title  string
}

func main() {

	router := gin.Default()
	tmpl := parseTemplates()

	images, _ := getImages("./pictures")

	galleryData := GalleryData{
		Images: images,
		Title:  "Gallery",
	}

	router.GET("/", func(c *gin.Context) {
		tmpl.ExecuteTemplate(c.Writer, "index.html", nil)
	})

	router.GET("/gallery", func(ctx *gin.Context) {
		tmpl.ExecuteTemplate(ctx.Writer, "gallery.html", galleryData)
	})

	router.GET("/upload", func(ctx *gin.Context) {
		tmpl.ExecuteTemplate(ctx.Writer, "upload.html", nil)
	})

	router.POST("/rescan", func(ctx *gin.Context) {
		var err2 error
		images, err2 = getImages("./pictures")
		if err2 != nil {
			log.Printf("Could not reload: %s", err2.Error())
			ctx.Status(400)
		}

		galleryData = GalleryData{
			Images: images,
			Title:  "Gallery",
		}
		ctx.Status(200)
	})

	router.GET("/settings", func(ctx *gin.Context) {
		tmpl.ExecuteTemplate(ctx.Writer, "settings.html", nil)
	})

	router.GET("/pic/*name", func(ctx *gin.Context) {
		name := ctx.Param("name")
		tmpl.ExecuteTemplate(ctx.Writer, "pic.html", Image{
			Name: name,
			Path: name,
		})
	})

	router.StaticFile("/output.css", "./templates/output.css")

	router.Static("/pictures", "./pictures")

	fmt.Println("Server running at http://localhost:3000")
	router.Run(":3000")
}

func getImageRec(dir string, images []Image) ([]Image, error) {
   	// Supported image extensions
	imageExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}
   
	// Read directory contents
	files, err := os.ReadDir(dir)
	if err != nil {
		return images, err
	}
   
	// Process each file
	for _, file := range files {
   
		name := file.Name()
		// Skip directories
		if file.IsDir() {
		    dirimages, err := getImages(filepath.Join(dir, name))
			if err != nil {
				return images, nil	
			}
			images = append(images, dirimages...)
			continue
		}
   
		if name[0] == '.' {
			continue
		}
   
		// Check if file has an image extension
		ext := strings.ToLower(filepath.Ext(name))
		if imageExtensions[ext] {
			// Create image object
			img := Image{
				Name: name,
				Path: filepath.Join(dir, name),
				URL:  filepath.Join(dir, name), // In a real app, this would be a web URL
			}
			images = append(images, img)
		}
	}
   
	return images, nil
}

// getImages reads all image files from a directory
func getImages(dir string) ([]Image, error) {
	var images []Image

	return getImageRec(dir, images)
}

func parseTemplates() *template.Template {
	templ := template.New("")
	err := filepath.Walk("./templates", func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, ".html") {
			_, err = templ.ParseFiles(path)
			if err != nil {
				log.Println(err)
			}
		}

		return err
	})

	if err != nil {
		panic(err)
	}

	return templ
}
