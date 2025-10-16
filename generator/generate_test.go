package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const generated_file_name = "output/generated_main.go"

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
			if id1 == id-1 {
				break
			}
			content = string(append([]byte(content[:id]), []byte(content[id1+1:])...))
		}
		for {
			id := strings.Index(content, "import \"")
			if id == -1 {
				break
			}
			id1 := id + strings.Index(content[id:], "\n")
			if id1 == id-1 {
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
		filesTotal := ""
		for _, file := range files {
			if file.pkg == first {
				packageFile += "//package " + first + "\n"
				packageFile += "//file " + file.name + "\n"
				packageFile += file.content
				filesTotal += file.content + "\n"
			}
		}

		for pkg := range packages {
			if strings.Contains(filesTotal, pkg+".") {
				if !toSkip[pkg] {
					fmt.Printf("adding package %s because used in %s\n", pkg, first)
					usedPackages = append(usedPackages, pkg)
					toSkip[pkg] = true
				}
			}
			packageFile = strings.Replace(packageFile, pkg+".", "", -1)
		}
		totalFile += packageFile
	}
	file, err := os.Create(generated_file_name)
	if err != nil {
		panic(err.Error())
	}

	src := []byte(totalFile)

	src, err = sanitizeCode(src)
	if err != nil {
		fmt.Fprintln(file, totalFile)
		err1 := file.Sync()
		if err1 != nil {
			panic(err1.Error())
		}
		panic(err.Error())
	}

	// DEBUG: Save file before RemoveGenerics
	os.MkdirAll("output/1", 0755)
	debugFile1, err := os.Create("output/1/before_remove_generics.go")
	if err == nil {
		fmt.Fprintln(debugFile1, string(src))
		debugFile1.Close()
	}

	src = RemoveGenerics(src)

	// DEBUG: Save file after RemoveGenerics
	os.MkdirAll("output/2", 0755)
	debugFile2, err := os.Create("output/2/after_remove_generics.go")
	if err == nil {
		fmt.Fprintln(debugFile2, string(src))
		debugFile2.Close()
	}

	src = RemoveUnusedCode(src)
	src, err = sanitizeCode(src)
	if err != nil {
		panic(err)
	}

	// DEBUG: Save file after RemoveUnusedCode
	os.MkdirAll("output/3", 0755)
	debugFile3, err := os.Create("output/3/after_remove_unused.go")
	if err == nil {
		fmt.Fprintln(debugFile3, string(src))
		debugFile3.Close()
	}

	src = RemoveUnsafeCasts(src)

	src, err = sanitizeCode(src)
	if err != nil {
		fmt.Fprintln(file, totalFile)
		err1 := file.Sync()
		if err1 != nil {
			panic(err1.Error())
		}
		panic(err.Error())
	}

	// DEBUG: Save final processed file
	os.MkdirAll("output/4", 0755)
	debugFile4, err := os.Create("output/4/final_processed.go")
	if err == nil {
		fmt.Fprintln(debugFile4, string(src))
		debugFile4.Close()
	}

	fmt.Fprintln(file, string(src))
}
