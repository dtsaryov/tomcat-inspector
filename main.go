package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Result[T any] struct {
	value    T
	error    bool
	errorMsg string
}

type ClassEntry struct {
	className   string
	zipLocation string
}

var CatalinaLocations = [...]string{"lib", "server/lib"}
var LibLocations = [...]string{"lib", "common/lib", "shared/lib"}

var Classes = [...][2]ClassEntry{
	{
		ClassEntry{"javax.servlet.jsp.JspPage", "javax/servlet/jsp/JspPage.class"},
		ClassEntry{"jakarta.servlet.jsp.JspPage", "jakarta/servlet/jsp/JspPage.class"},
	},
	{
		ClassEntry{"javax.servlet.Servlet", "javax/servlet/Servlet.class"},
		ClassEntry{"jakarta.servlet.Servlet", "jakarta/servlet/Servlet.class"},
	},
}

func main() {
	if len(os.Args) == 1 {
		printResult(createErrorResult("No command provided"))
		return
	}
	if len(os.Args) == 2 {
		printResult(createErrorResult("Tomcat path is not provided"))
		return
	}

	switch os.Args[1] {
	case "getServerInfo":
		{
			printResult(getServerInfo(strings.TrimSuffix(os.Args[2], "/")))
		}
	case "searchForClasses":
		{
			result := searchForClasses(strings.TrimSuffix(os.Args[2], "/"))
			if result.error {
				printResult(createErrorResult(result.errorMsg))
				return
			}

			var classes = result.value
			if !isSearchDone(classes) {
				printResult(createErrorResult("Unable to find all required classes"))
				return
			}

			for class, jar := range classes {
				fmt.Printf("%s:%s\n", class, jar)
			}
		}
	default:
		printResult(createErrorResult(fmt.Sprintf("Command is not supported: %s", os.Args[1])))
	}
}

func getServerInfo(tomcatHome string) Result[string] {
	var catalinaJar = findCatalinaJar(tomcatHome)
	if len(catalinaJar) == 0 {
		return createErrorResult(fmt.Sprintf("Unable to find catalina.jar. Tomcat path: %s", tomcatHome))
	}

	catalinaJarReader, err := zip.OpenReader(catalinaJar)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to read %s", catalinaJar))
	}
	defer func(catalinaJarReader *zip.ReadCloser) {
		err := catalinaJarReader.Close()
		if err != nil {

			panic(err)
		}
	}(catalinaJarReader)

	var serverInfo = ""

	for _, file := range catalinaJarReader.File {
		if strings.HasSuffix(file.Name, "ServerInfo.properties") {
			fileReader, err := file.Open()
			if err != nil {
				return createErrorResult(fmt.Sprintf("Failed to read %s", file.Name))
			}

			scanner := bufio.NewScanner(fileReader)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				var property = scanner.Text()
				if strings.HasPrefix(property, "server.info") {
					serverInfo = strings.SplitAfter(property, "=")[1]
				}
			}

			err = fileReader.Close()
			if err != nil {
				panic(err)
			}
		}
	}

	if len(serverInfo) > 0 {
		return Result[string]{serverInfo, false, ""}
	} else {
		return createErrorResult("Unable to find \"server.info\" property")
	}
}

func findCatalinaJar(tomcatHome string) string {
	for _, path := range CatalinaLocations {
		files, err := os.ReadDir(tomcatHome + "/" + path)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.Name() == "catalina.jar" {
				return tomcatHome + "/" + path + "/catalina.jar"
			}
		}
	}
	return ""
}

func searchForClasses(tomcatHome string) Result[map[string]string] {
	var classes = make(map[string]string)

	var jars = collectJars(tomcatHome)
	if len(jars) == 0 {
		return createMapResult(classes)
	}

	for _, path := range jars {
		if isSearchDone(classes) {
			return createMapResult(classes)
		}

		r, err := zip.OpenReader(path)
		if err != nil {
			continue
		}

		for _, file := range r.File {
			if isSearchDone(classes) {
				break
			}

			for _, classEntries := range Classes {
				if isSearchDone(classes) {
					break
				}

				for _, classEntry := range classEntries {
					if isSearchDone(classes) {
						break
					}

					if file.Name == classEntry.zipLocation {
						classes[classEntry.className] = path
					}
				}
			}
		}

		err = r.Close()
		if err != nil {
			panic(err)
		}
	}

	return createMapResult(classes)
}

func collectJars(tomcatHome string) []string {
	var jars []string
	for _, path := range LibLocations {
		files, err := os.ReadDir(tomcatHome + "/" + path)
		if err != nil {
			continue
		}

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".jar") {
				jars = append(jars, tomcatHome+"/"+path+"/"+file.Name())
			}
		}
	}
	return jars
}

func isSearchDone(classes map[string]string) bool {
	for _, classVariants := range Classes {
		var classFound = false

		for _, classVariant := range classVariants {
			var _, hasClass = classes[classVariant.className]
			if hasClass {
				classFound = true
				break
			}
		}

		if !classFound {
			return false
		}
	}
	return true
}

func printResult(result Result[string]) {
	if result.error {
		_, err := fmt.Fprint(os.Stderr, result.errorMsg)
		if err != nil {
			panic(err)
		}
	} else {
		_, err := fmt.Fprintf(os.Stdout, result.value)
		if err != nil {
			panic(err)
		}
	}
}

func createMapResult(value map[string]string) Result[map[string]string] {
	return Result[map[string]string]{value, false, ""}
}

func createErrorResult(errorMsg string) Result[string] {
	return Result[string]{"", true, errorMsg}
}
