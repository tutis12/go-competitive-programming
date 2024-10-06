package main

import (
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
)

type fileInfo struct {
	pkg     string
	content string
	name    string
}

func TestMergeEverything(*testing.T) {
	var files []fileInfo
	dirNames := []string{"../"}
	for len(dirNames) != 0 {
		dirName := dirNames[0]
		dirNames = dirNames[1:]
		dir, err := os.ReadDir(dirName)
		if err != nil {
			panic(err.Error())
		}
		for _, file := range dir {
			filePath := dirName + "/" + file.Name()
			if strings.Contains(filePath, "..//generator") {
				continue
			}
			if file.IsDir() {
				dirNames = append(dirNames, filePath)
			} else {
				_, ok := strings.CutSuffix(file.Name(), ".go")
				if !ok {
					continue
				}
				file, err := os.Open(filePath)
				if err != nil {
					panic(err.Error())
				}
				buffer := make([]byte, 100000)
				n, err := file.Read(buffer)
				if err != nil {
					panic(err.Error())
				}
				if n == len(buffer) {
					panic("file too big")
				}
				files = append(files, fileInfo{
					content: string(buffer[:n]),
					name:    file.Name(),
				})
			}
		}
	}
	for i, fileInfo := range files {
		file := fileInfo.content

		file, ok := strings.CutPrefix(file, "package ")
		if !ok {
			panic("file without package")
		}
		id := strings.IndexByte(file, '\n')
		files[i].pkg = file[:id]
		content := file[id+1:]
		for {
			id := strings.Index(content, "import (")
			if id == -1 {
				break
			}
			id1 := id + strings.Index(content[id:], ")")
			if id1 == -1 {
				break
			}
			content = string(append([]byte(content[:id]), []byte(content[id1+1:])...))
		}
		files[i].content = content
	}
	packages := make(map[string]struct{})
	for _, file := range files {
		packages[file.pkg] = struct{}{}
	}
	usedPackages := []string{"main"}
	totalFile := "package main\nimport (\n\"fmt\"\n\"math\"\n\"os\"\n\"runtime\"\n\"strconv\"\n\"sync\"\n\"sync/atomic\"\n\"time\"\n)\n"
	toSkip := make(map[string]bool)
	toSkip["main"] = true
	for len(usedPackages) != 0 {
		first := usedPackages[0]
		usedPackages = usedPackages[1:]
		packageFile := ""
		for _, file := range files {
			if file.pkg == first {
				packageFile += "//package " + first + "\n"
				packageFile += "//file " + file.name + "\n"
				packageFile += file.content
			}
		}
		for pkg := range packages {
			if strings.Contains(packageFile, pkg+".") {
				if !toSkip[pkg] {
					usedPackages = append(usedPackages, pkg)
					toSkip[pkg] = true
				}
			}
			packageFile = strings.Replace(packageFile, pkg+".", "", -1)
		}
		totalFile += packageFile
	}
	file, err := os.Create("main.go")
	if err != nil {
		panic(err.Error())
	}
	totalFileBytes, err := format.Source([]byte(totalFile))
	if err != nil {
		fmt.Fprintln(file, totalFile)
		file.Sync()
		panic(err.Error())
	}
	fmt.Fprintln(file, string(totalFileBytes))
}
