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

// Maybe: mgit all status --summary -> summary of all directories locally
// Maybe: execute in specific directories from the multiplexer

var userConfigDir, err = os.UserConfigDir()
var configPath = filepath.Join(userConfigDir, "mgit", "mgit_config.json")
var msg string = `
Supported commands:
mgit help                       -> see this message
mgit alias "repo_name" "alias"  -> take the name of a directory and give it an alias for ease of use
mgit git_repo/alias git_command -> run git command in the desired directory
`

func resolveAbsolutePath(path string, aliases map[string]string) string {
	if path, ok := aliases[path]; ok {
		return path
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil || !fileInfo.IsDir() {
		fmt.Printf("%s is not a valid directory.\n", path)
		os.Exit(1)
	}

	return absPath
}

func runGitCommand(path string, args []string, wg *sync.WaitGroup) {
	defer wg.Done()

	cmd := exec.Command("git", args...)
	cmd.Dir = path

	output, err := cmd.CombinedOutput()

	fmt.Printf("\n--- %s ---\n", path)
	fmt.Print(string(output))

	if err != nil {
		fmt.Println("Git command failed in repository", path)
		fmt.Println(err)
	}
}

func checkValidDir(path string) bool {
	cmd := exec.Command(
		"git",
		"-C", path,
		"rev-parse",
		"--is-inside-work-tree",
	)

	output, err := cmd.Output()

	return err == nil && strings.TrimSpace(string(output)) == "true"
}

func saveAliases(aliases map[string]string) {
	a, _ := json.Marshal(aliases)
	os.WriteFile(configPath, a, 0644)
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

	aliases := make(map[string]string)
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
	if len(os.Args) < 2 {
		fmt.Println(msg)
		return
	}
	var s string = os.Args[1]

	if s == "help" {
		fmt.Println(msg)
	} else if s == "alias" {
		if len(os.Args) != 4 {
			fmt.Println("\nToo many arguments given.\n", msg)
			return
		}
		aliases[os.Args[2]] = resolveAbsolutePath(os.Args[3], aliases)
		saveAliases(aliases)
	} else if s == "unalias" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: mgit unalias <alias>")
		}
		delete(aliases, os.Args[2])
		saveAliases(aliases)
	} else if s == "list" {
		fmt.Println("\nAliases:")
		fmt.Println("────────────────────────────────────────────")

		for alias, path := range aliases {
			fmt.Printf("%-20s %s\n", alias, path)
		}
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
			var a []string = strings.Split(s, ",")
			for _, v := range a {
				wg.Add(1)
				go runGitCommand(resolveAbsolutePath(v, aliases), os.Args[2:], &wg)
			}
			wg.Wait()
		} else {
			s = resolveAbsolutePath(s, aliases)
			wg.Add(1)
			go runGitCommand(s, os.Args[2:], &wg)
			wg.Wait()
		}
	}
}
