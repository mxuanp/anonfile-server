package model

import(
	"github.com/jinzhu/gorm"
)

type File struct {
	gorm.Model
	Name string
	Fullname string
	Size string
	Category string
	Parent string //使用树级目录设计
	Url string
}

func (file File) String() string {
	return "[" + file.Name + "," + file.Size + "," + file.Category + "," + file.Parent+ "," + file.Url + "]"
}

type FileSlice []File

func (slice FileSlice) Len() int {
	return len(slice)
}

func (slice FileSlice) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func (slice FileSlice) Less(i, j int) bool {
	return slice[i].Name < slice[j].Name
}
