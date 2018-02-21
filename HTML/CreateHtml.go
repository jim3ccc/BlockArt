package HTML

import (
	"os"
	"fmt"
	"io/ioutil"
	"strconv"
)

func CreateFile(fname string) {
	path :=  fname
	_, err := os.Stat(path)
	fmt.Println(err)

	// create file if not exists
	if os.IsNotExist(err) {
		file, err := os.Create(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		file.Close()
	}
}

func ReadFile( fname string) string {
	// re-open file
	path :=  fname
	dat, err := ioutil.ReadFile(path)
	fmt.Print(err)
	return string(dat)

}

func WriteFile( fname, content string) error {
	// open file using READ & WRITE permission
	path :=  fname
	d1 := []byte(content)
	err := ioutil.WriteFile(path, d1, 0644)
	return err
}

func makeSvgHeader(width, height uint32)string{
	h := strconv.FormatUint(uint64(height), 10)
	w := strconv.FormatUint(uint64(width), 10)
	return "<svg height=\"" + h+ "\" width =\"" +w +"\">"
}

var htmlHeaders = "<!DOCTYPE html>\n<html>\n<body>"
var svgEnd = "</svg>"
var htmlFooter = "\n</body>\n</html>"
var testString = "<ellipse cx=\"240\" cy=\"100\" rx=\"220\" ry=\"30\" style=\"fill:purple\" />"
var teststr = []string{testString,testString,testString}

func CreateHtml(svgStrings []string, width, height uint32, name string){
	Content := htmlHeaders
	CreateFile(name)
	svgStart := makeSvgHeader(width,height)
	Content+= svgStart

	for _, svg :=range svgStrings{
		Content+="\n" +svg
	}
	//Content+=svgEnd
	Content+=htmlFooter
	WriteFile(name, Content)
	fmt.Println("Creating HTML: ", name)
}

//func main(){
//	CreateHtml(teststr, 5000,5000)
//}
