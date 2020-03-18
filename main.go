package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/mxuanp/anonfile-go-server/model"
	"github.com/mxuanp/anonfile-go-server/utils"
)

var (
	Conf map[string]string

	db *gorm.DB
)

func init() {
	f, err := os.Open("config.json")
	defer f.Close()

	if err != nil {
		log.Fatal(err)
	}

	json.NewDecoder(f).Decode(&Conf)

	db, err = gorm.Open("sqlite3", "file.db")
	if err != nil {
		log.Println(err)
	}
	//设置数据库连接池
	db.DB().SetMaxIdleConns(5)
	db.DB().SetMaxOpenConns(10)
	db.DB().SetConnMaxLifetime(time.Hour)

}

func main() {
	defer db.Close()

	db.AutoMigrate(&model.File{})
	//check root dir
	var count int
	db.Where("name = ?", "/").First(&model.File{}).Count(&count)
	if count == 0 {
		db.Create(&model.File{Name: "/", Size: "4KB", Category: "directory", Parent: "", Url: ""})
	}

	router := gin.Default()

	//上传文件, 或创建文件夹
	router.POST("/api/file/new", func(c *gin.Context) {
		flag = c.PostForm("flag")
		parent := c.PostForm("parent")
		form, _ := c.MultipartForm()
		files := form.File["files"]

		if !CheckParent(parent) {
			c.JSON(200, gin.H{
				"status":  "3003",
				"message": "parent dir is not exist",
			})
			return
		}

		if flag == "directory" {
			dirName = c.PostForm("dirName")
			if CheckFile(dirName, parent) {
				c.JSON(200, gin.H{
					"status":  "3003",
					"message": "dir is exist",
				})
				return
			}

			file := model.File{Name: dirName, Size: "4KB", Category: "directory", Parent: parent, Url: ""}
			db.Create(&file)

			c.JSON(200, gin.H{
				"status":  "2002",
				"message": "create successed",
				"data":    file,
			})

			return
		}

		//开启事务, 有上传失败，放弃所有上传文件，返回结果
		tx := db.Begin()

		for _, file := range files {
			if CheckFile(file.Filename, parent) {
				file.Filename = utils.RandString(4) + "_" + file.Filename
			}

			fields := map[string]string{
				"filename": file.Filename,
			}

			if token, ok := Conf["token"]; ok {
				fields["token"] = token
			}

			f, err := file.Open()
			if err != nil {
				fmt.Println(err)
				continue
			}

			resp, err := UploadFile(Conf["upload_url"], f, fields)
			if err != nil {
				fmt.Println(err)
			}

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)

			if resp.StatusCode == http.StatusOK {
				info := utils.ParseSuccessRes(body)
				tx.Create(&model.File{Name: file.Filename, Size: info.Data.File.Metadata.Size.Readable, Category: "file", Parent: parent, Url: info.Data.File.Url.Full})

			} else {
				tx.Rollback()

				info := utils.ParseErrorRes(body)
				c.JSON(200, gin.H{
					"status":  info.Err.Code,
					"message": info.Err.Message,
				})
				return
			}

		}

		tx.Commit()

		c.JSON(200, gin.H{
			"status":  "2002",
			"message": "upload successed",
		})
	})

	//获取文件夹信息, 以及该文件夹下所有文件
	router.GET("/api/file/info/*fileName", func(c *gin.Context) {
		file := c.Param("fileName")

		//对路径的检查到此，以后逐渐增加其它的路径合法性检查，比如，路径像这样的 adb/adv是不合法的

		files, isFile := GetFiles(file)

		c.JSON(200, gin.H{
			"status":  "2002",
			"message": "get successed",
			"data": {
				"files":  files,
				"isFile": isFile,
			},
		})
	})

	//更新文件信息
	router.PUT("/api/file/update", func(c *gin.Context) {
		fileName := c.PostForm("fileName")

		id, err := strconv.Atoi(c.PostForm("id"))
		if err != nil {
			c.JSON(200, gin.H{
				"status":  "3004",
				"message": "parameter error",
			})
		}

		file := model.File{Model: gorm.Model{ID: uint(id)}}
		db.First(&file)

		if CheckFile(fileName, file.Parent) {
			c.JSON(200, gin.H{
				"status":  "3004",
				"message": "name was exist",
			})
			return
		}

		if file.Category == "directory" { //要修改的file是文件夹类型，需要同时修改该目录下所有文件的父级目录 parent
			db.Model(model.File{}).Where("parent = ?", file.Parent+"/"+file.Name).Updates(model.File{Parent: file.Parent + "/" + fileName})
		}
		file.Name = fileName
		db.Save(&file)

		c.JSON(200, gin.H{
			"status":  "2002",
			"message": "update successed",
			"data":    file,
		})
	})

	//删除文件或文件夹
	router.DELETE("/api/file/*fileName", func(c *gin.Context) {
		path := strings.Split(c.Param("fileName"), "/")
		fileName := path[len(path)-1]
		parent := strings.Join(path[:len(path)-1], "/")

		var file model.File
		db.Where("parent = ? and name = ?", parent, fileName).First(&file)

		if file.Category == "directory" {
			db.Unscoped().Where("parent = ?", path).Delete(model.File{})
		}

		db.Unscoped().Delete(&file) //gorm.Model定义了deleted_at, 直接Delete会进行软删除，使用Unscoped设置直接删除记录

		c.JSON(200, gin.H{
			"status":  "2002",
			"message": "delete successed",
		})
	})

	//给客户端测试下联通情况
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	router.Run()
}

//GetFiles 查询当前文件夹下所有文件，parent为当前文件夹名
func GetFiles(name string) (files []model.File, isFile bool) {
	parent := "NULL"
	filename := name
	if name != "/" {
		paths := strings.Split(name, "/")
		filename = paths[len(paths)-1]
		parent = strings.Join(paths[:len(paths)-1], "/")
	}

	var file model.File
	db.Where("parent = ? and name = ", parent, filename).First(&file)

	if file.Category == "file" {
		files = append(files, file)
		isFile = true
		return
	}

	db.Where("parent = ?", name).Find(&files)
	sort.Sort(model.FileSlice(files))
	isFile = false
	return
}

func CheckParent(parent string) bool {
	if parent == "/" {
		return true
	}
	var count int
	paths := strings.Split(parent, "/")
	return CheckFile(paths[len(paths)-1], strings.Join(paths[:len(paths)-1], "/"))
}

//CheckFile 检查文件是否已存在
func CheckFile(name, parent string) bool {
	var count int
	db.Where("parent = ? and name = ?", parent, name).First(&model.File{}).Count(&count)
	return count != 0
}

//上传文件
func UploadFile(url string, f io.Reader, fields map[string]string) (*http.Response, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("file", fields["filename"])
	if err != nil {
		return nil, fmt.Errorf("CreateFromFile %v", err)
	}

	_, err = io.Copy(fw, f)
	if err != nil {
		return nil, fmt.Errorf("coping fileWriter %v", err)
	}

	for k, v := range fields {
		writer.WriteField(k, v)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("writerClose %v", err)
	}

	resp, err := http.Post(url, writer.FormDataContentType(), body)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
