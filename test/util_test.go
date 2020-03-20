package anonfile_test

import(
	"testing"
	"github.com/mxuanp/anonfile-server/utils"
)

func TestRandString(t *testing.T){
	str := utils.RandString(4)
	println(str)
}
