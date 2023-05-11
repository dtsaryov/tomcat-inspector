package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"os"
	"os/exec"
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
	if len(os.Args) == 1 {
		handleResult("Error: No command provided")
		return
	}

	switch os.Args[1] {
	case "getServerInfo":
		{
			if len(os.Args) == 2 {
				handleResult("Error: Tomcat path is not provided")
				return
			}

			handleResult(getServerInfo(strings.TrimSuffix(os.Args[2], "/")))
		}
	case "searchForClasses":
		{
			if len(os.Args) == 2 {
				handleResult("Error: Tomcat path is not provided")
				return
			}

			var tomcatHome = strings.TrimSuffix(os.Args[2], "/")
			var classes = searchForClasses(tomcatHome)
			if !isSearchDone(classes) {
				handleResult(fmt.Sprintf("Error: unable to find all required classes"))
				return
			}

			for class, jar := range searchForClasses(tomcatHome) {
				fmt.Printf("%s:%s\n", class, jar)
			}
		}
	case "getJavaHome":
		{
			handleResult(findJavaHome())
		}
	default:
		handleResult(fmt.Sprintf("Error: Command is not supported: %s", os.Args[1]))
	}
}

func getServerInfo(tomcatHome string) string {
	var catalinaJar = findCatalinaJar(tomcatHome)
	if len(catalinaJar) == 0 {
		return fmt.Sprintf("Error: Unable to find catalina.jar. Tomcat path: %s", tomcatHome)
	}

	reader, err := zip.OpenReader(catalinaJar)
	if err != nil {
		return fmt.Sprintf("Error: Failed to read %s", catalinaJar)
	}

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "ServerInfo.properties") {
			readFile, err := file.Open()
			if err != nil {
				return fmt.Sprintf("Error: Failed to read %s", file.Name)
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

func findJavaHome() string {
	javaHomeEnv, exists := os.LookupEnv("JAVA_HOME")
	if exists {
		return javaHomeEnv
	}

	output, err := exec.Command("which", "java").Output()
	if err != nil {
		return "Error: Failed to execute \"which java\""
	}

	if len(output) > 0 {
		return strings.Replace(strings.TrimSuffix(string(output), "\n"), "/bin/java", "", 1)
	}

	return "Error: Unable to detect JAVA_HOME"
}

//goland:noinspection GoUnhandledErrorResult
func handleResult(result string) {
	if strings.HasPrefix(result, "Error: ") {
		fmt.Fprintf(os.Stderr, strings.Replace(result, "Error: ", "", 1))
	} else {
		fmt.Fprintf(os.Stdout, "%s", result)
	}
}
