package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var userConfigDir, err = os.UserConfigDir()
var configPath = filepath.Join(userConfigDir, "mgit", "mgit_config.json")
var msg string = `
Supported commands:
mgit help                       -> see this message
mgit alias "repo_name" "alias"  -> take the name of a directory and give it an alias for ease of use
mgit git_repo/alias git_command -> run git command in the desired directory
`

func runGitCommand(path string, args []string, wg *sync.WaitGroup) {
	defer wg.Done()

	cmd := exec.Command("git", args...)
	cmd.Dir = path

	output, err := cmd.CombinedOutput()

	fmt.Printf("\n--- %s ---\n", path)
	fmt.Print(string(output))

	if err != nil {
		panic(err)
	}
}

func checkValidDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	if fileInfo.IsDir() {
		fileInfo, err = os.Stat(filepath.Join(path, ".git"))
		if err != nil {
			// does not exist
			return false
		}
		return true
	}
	return false
}

func saveAliases(aliases map[string]string) {
	a, _ := json.Marshal(aliases)
	os.WriteFile(configPath, a, os.FileMode(os.O_TRUNC))
}

func main() {

	var wg sync.WaitGroup

	// Load config file
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(filepath.Join(userConfigDir, "mgit"), 0755)
	if err != nil {
		panic(err)
	}
	_, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	var aliases map[string]string
	data, err := os.ReadFile(configPath)
	err = json.Unmarshal(data, &aliases)

	// index CWD
	cwd, _ := os.Getwd()
	subfolders, _ := os.ReadDir(cwd)
	var dirs []string
	for i := 0; i < len(subfolders); i++ {
		if checkValidDir(filepath.Join(cwd, subfolders[i].Name())) {
			dirs = append(dirs, filepath.Join(cwd, subfolders[i].Name()))
		}
	}

	// Parse CLI
	var s string = os.Args[1]

	if s == "help" {
		fmt.Println(msg)
	} else if s == "alias" {
		if len(os.Args) != 4 {
			fmt.Println("\nToo many arguments given.\n", msg)
			return
		}
		for i := 2; i < len(os.Args); i++ {
			aliases[os.Args[3]] = os.Args[4]
		}
		saveAliases(aliases)
	} else {
		// treat actual git commands
		if s == "all" {
			for i := 0; i < len(dirs); i++ {
				var repo = dirs[i]
				wg.Add(1)
				go runGitCommand(repo, os.Args[2:], &wg)
			}
			wg.Wait()
		} else if strings.Contains(s, ",") {
			// treat multiple repos at the same time
			// var a []string = strings.Split(s, ",")
			// for i : range len(a) {
			// 	if aliases[i] != nil {
			// 		go runGitCommand(aliases[i], os.Args[2:], &wg)
			// 	} else {

			// 	}
			// }
		}
	}
}
