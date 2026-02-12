package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
)

// Image represents a single image file
type Image struct {
	Name string
	Path string
}

type GalleryData struct {
	Images []Image
	Title  string
}

func main() {

	db := connectDb()
	defer db.Close()
	initDb(db)

	router := gin.Default()
	tmpl := parseTemplates()

	images, _ := loadImages(db)

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
		images, err2 = scanImages("./pictures", db)
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

func scanImageRec(dir string, images []Image, db *sql.DB) ([]Image, error) {
	imageExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return images, err
	}

	for _, file := range files {

		name := file.Name()
		if file.IsDir() {
			dirimages, err := scanImages(filepath.Join(dir, name), db)
			if err != nil {
				return images, nil
			}
			images = append(images, dirimages...)
			continue
		}

		if name[0] == '.' {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		if imageExtensions[ext] {
			img := Image{
				Name: name,
				Path: filepath.Join(dir, name),
			}

			_, err := db.Exec(`
			        INSERT OR IGNORE INTO pictures (path) VALUES (?)

			`, filepath.Join(dir, name))
			if err != nil {
				panic(err)
			}

			images = append(images, img)
		}
	}

	return images, nil
}

func scanImages(dir string, db *sql.DB) ([]Image, error) {
	var images []Image

	return scanImageRec(dir, images, db)
}

func loadImages(db *sql.DB) ([]Image, error) {
	var images []Image

	rows, err := db.Query("SELECT path from pictures;")
	if err != nil {
        log.Printf("Error while loading pictures: %s", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var image Image
		var path string
		if err := rows.Scan(&path); err != nil {
		    log.Printf("Error: %s", err.Error())
		}
		image.Path = path
		split := strings.Split(path, "/")
		image.Name = split[len(split)-1]
		images = append(images, image)
	}

	return images, err
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

func connectDb() *sql.DB {
	db, err := sql.Open("sqlite3", "pictures.db")
	if err != nil {
		log.Panic(err)
	}

	return db
}

func initDb(db *sql.DB) {

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS pictures (
            id INTEGER PRIMARY KEY,
            path TEXT NOT NULL,
            UNIQUE(path)
            );`)
	if err != nil {
		log.Printf("Error: %s", err.Error())
	} else {
		log.Printf("Result: No error")
	}

	scanImages("./pictures", db)
}
