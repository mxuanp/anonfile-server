package utils

import (
	"encoding/json"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

//解析上传结果
func ParseSuccessRes(res []byte) (sr SuccessRes) {
	json.Unmarshal(res, &sr)
	return
}

//解析错误结果
func ParseErrorRes(res []byte) (er ErrorRes) {
	json.Unmarshal(res, &er)
	return
}

type (
	/** 成功返回结果
	  {
	      "status": true,
	      "data": {
	          "file": {
	              "url": {
	                  "full": "https://anonfile.com/u1C0ebc4b0/file.txt",
	                  "short": "https://anonfile.com/u1C0ebc4b0"
	              },
	              "metadata": {
	                  "id": "u1C0ebc4b0",
	                  "name": "file.txt",
	                  "size": {
	                      "bytes": 6861,
	                      "readable": "6.7 KB"
	                  }
	              }
	          }
	      }
	  }
	*/
	SuccessRes struct {
		Status bool `json:"status"`
		Data   Data `json:"data"`
	}

	Data struct {
		File FileInfo `json:"file"`
	}

	FileInfo struct {
		Url      Url      `json:"url"`
		Metadata Metadata `json:"metadata"`
	}

	Url struct {
		Full  string `json:"full"`
		Short string `json:"short"`
	}

	Metadata struct {
		Id   string `json:"id"`
		Name string `json:"name"`
		Size Size   `json:"size"`
	}

	Size struct {
		Bytes    int    `json:"bytes"`
		Readable string `json:"readable"`
	}

	/* 错误返回结果
	   {
	       "status": false,
	       "error": {
	           "message": "The file is too large. Max filesize: 5 GiB",
	           "type": "ERROR_FILE_SIZE_EXCEEDED",
	           "code": 31
	       }
	   }
	*/

	ErrorRes struct {
		Status bool  `json:"status"`
		Err    Error `json:"error"`
	}

	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	}
)

//生成随机字符串
func RandString(n int) string {
	const letter = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
