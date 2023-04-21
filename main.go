package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"os"
	"strings"
)

type ClassEntry struct {
	className   string
	zipLocation string
}
type ClassEntries struct {
	entries []ClassEntry
}

var CatalinaLocations = [...]string{"lib", "server/lib"}
var LibLocations = [...]string{"lib", "common/lib", "shared/lib"}

var Classes = [2]ClassEntries{
	{
		[]ClassEntry{
			{"javax.servlet.jsp.JspPage", "javax/servlet/jsp/JspPage.class"},
			{"jakarta.servlet.jsp.JspPage", "jakarta/servlet/jsp/JspPage.class"},
		},
	},
	{
		[]ClassEntry{
			{"javax.servlet.Servlet", "javax/servlet/Servlet.class"},
			{"jakarta.servlet.Servlet", "jakarta/servlet/Servlet.class"},
		},
	},
}

func main() {
	var cmd = os.Args[1]
	var tomcatHome = strings.TrimSuffix(os.Args[2], "/")

	switch cmd {
	case "getServerInfo":
		{
			println(readProperty(tomcatHome))
		}
	case "searchForClasses":
		{
			for class, jar := range searchForClasses(tomcatHome) {
				fmt.Printf("%s:%s\n", class, jar)
			}
		}
	default:
		println("Error: No command specified")
	}
}

func readProperty(tomcatHome string) (serverInfo string) {
	var catalinaJar = findCatalinaJar(tomcatHome)
	if len(catalinaJar) == 0 {
		return "Error: Unable to find catalina.jar"
	}

	reader, err := zip.OpenReader(catalinaJar)
	if err != nil {
		return "Error: Failed to read catalina.jar"
	}

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "ServerInfo.properties") {
			readFile, err := file.Open()
			if err != nil {
				return "Error: Failed to read ServerInfo.properties"
			}

			scanner := bufio.NewScanner(readFile)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				var property = scanner.Text()
				if strings.HasPrefix(property, "server.info") {
					return strings.SplitAfter(property, "=")[1]
				}
			}
		}
	}

	return "Error: Unable to find \"server.info\" property"
}

func findCatalinaJar(tomcatHome string) (catalinaJarLocation string) {
	for _, path := range CatalinaLocations {
		files, _ := os.ReadDir(tomcatHome + "/" + path)

		for _, file := range files {
			if file.Name() == "catalina.jar" {
				return tomcatHome + "/" + path + "/catalina.jar"
			}
		}
	}
	return ""
}

func searchForClasses(tomcatHome string) map[string]string {
	var classes = make(map[string]string)

	var jars = collectJars(tomcatHome)
	if len(jars) == 0 {
		println("Error: No JARs found")
		return classes
	}

	for _, path := range jars {
		if isSearchDone(classes) {
			return classes
		}

		reader, err := zip.OpenReader(path)
		if err != nil {
			continue
		}

		for _, file := range reader.File {
			if isSearchDone(classes) {
				return classes
			}

			for _, classEntries := range Classes {
				if isSearchDone(classes) {
					return classes
				}

				for _, classEntry := range classEntries.entries {
					if isSearchDone(classes) {
						return classes
					}

					if file.Name == classEntry.zipLocation {
						classes[classEntry.className] = path
					}
				}
			}
		}
	}

	return classes
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

		for _, classVariant := range classVariants.entries {
			var _, hasClass = classes[classVariant.className]
			classFound = hasClass
		}

		if !classFound {
			return false
		}
	}
	return true
}
