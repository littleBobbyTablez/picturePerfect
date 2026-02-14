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

type Image struct {
	Name string
	Path string
}

type Directory struct {
	Name string
	Path string
}

type GalleryData struct {
	Images      []Image
	Directories []Directory
	Title       string
	Parent      string
	IsBase      bool
}

func main() {

	db := connectDb()
	defer db.Close()
	initDb(db)

	router := gin.Default()
	tmpl := parseTemplates()

	galleryData := GalleryData{
		Images:      nil,
		Directories: nil,
		Title:       "Gallery",
		Parent:      "/pictures",
		IsBase:      true,
	}

	router.GET("/", func(c *gin.Context) {
		tmpl.ExecuteTemplate(c.Writer, "index.html", nil)
	})

	router.GET("/gallery/*dir", func(ctx *gin.Context) {
		dir := ctx.Param("dir")

		images, subdirs, err := loadImages(dir, db)
		if err != nil {
			log.Fatal(err)
		}

		base := dir == "/pictures"

		galleryData = GalleryData{
			Images:      images,
			Directories: subdirs,
			Title:       dir,
			Parent:      filepath.Dir(dir),
			IsBase:      base,
		}
		tmpl.ExecuteTemplate(ctx.Writer, "gallery.html", galleryData)
	})

	router.GET("/upload", func(ctx *gin.Context) {
		tmpl.ExecuteTemplate(ctx.Writer, "upload.html", nil)
	})

	router.POST("/rescan", func(ctx *gin.Context) {
		var err2 error
		galleryData.Images, galleryData.Directories, err2 = scanImages("pictures", db)
		if err2 != nil {
			log.Printf("Could not reload: %s", err2.Error())
			ctx.Status(400)
		}

		ctx.Status(200)
	})

	router.GET("/settings", func(ctx *gin.Context) {
		tmpl.ExecuteTemplate(ctx.Writer, "settings.html", nil)
	})

	router.GET("/pic/*path", func(ctx *gin.Context) {
		path := ctx.Param("path")
		name := filepath.Base(path)
		dir := filepath.Dir(path)
		tmpl.ExecuteTemplate(ctx.Writer, "pic.html", Image{
			Name: name,
			Path: dir,
		})
	})

	router.StaticFile("/output.css", "./templates/output.css")

	router.Static("/pictures", "./pictures")

	fmt.Println("Server running at http://localhost:3000")
	router.Run(":3000")
}

func scanImageRec(dir string, images []Image, directories []Directory, db *sql.DB) ([]Image, []Directory, error) {
	imageExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}

	files, err := os.ReadDir("." + dir)
	if err != nil {
		return images, directories, err
	}

	for _, file := range files {

		name := file.Name()
		if file.IsDir() {

			directory := Directory{
				Name: file.Name(),
				Path: dir,
			}
			directories = append(directories, directory)
			insertImageOrDir(name, dir, true, db)

			subpath := filepath.Join(dir, name)

			dirimages, subdirectories, err := scanImages(subpath, db)
			if err != nil {
				return images, directories, nil
			}
			directories = append(directories, subdirectories...)
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
				Path: dir,
			}

			insertImageOrDir(name, dir, false, db)

			images = append(images, img)
		}
	}

	return images, directories, nil
}

func scanImages(dir string, db *sql.DB) ([]Image, []Directory, error) {
	var images []Image
	var directory []Directory

	return scanImageRec(dir, images, directory, db)
}

func insertImageOrDir(name string, dir string, isDir bool, db *sql.DB) {
	_, err := db.Exec(`
			        INSERT OR IGNORE INTO pictures (path, name, isDir) VALUES (?, ?, ?)

			`, dir, name, isDir)
	if err != nil {
		panic(err)
	}
}

func loadImages(loadPath string, db *sql.DB) ([]Image, []Directory, error) {
	var images []Image
	var directories []Directory

	q := fmt.Sprintf("SELECT path, name, isDir from pictures where path='%s';", loadPath)
	rows, err := db.Query(q)
	if err != nil {
		log.Printf("Error while loading pictures: %s", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var name string
		var isDir bool
		if err := rows.Scan(&path, &name, &isDir); err != nil {
			log.Printf("Error: %s", err.Error())
		}
		if isDir {
			var directory Directory
			directory.Name = name
			directory.Path = path
			directories = append(directories, directory)
		} else {
			var image Image
			image.Path = path
			image.Name = name
			images = append(images, image)

		}
	}

	return images, directories, err
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
            name TEXT NOT NULL,
            isDir INTEGER NOT NULL,
            UNIQUE(path, name)
            );`)
	if err != nil {
		log.Printf("Error: %s", err.Error())
	} else {
		log.Printf("Result: No error")
	}

	scanImages("/pictures", db)
}
