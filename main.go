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
var aliasConfigPath = filepath.Join(userConfigDir, "mgit", "mgit_alias_config.json")
var shortcutConfigPath = filepath.Join(userConfigDir, "mgit", "mgit_shortcut_config.json")

var msg string = `
MGit — Git multiplexer & command launcher

USAGE

mgit <repo> <git-command> [args...]
mgit all <git-command> [args...]
mgit <repo1,repo2,...> <git-command> [args...]

Run Git commands in one or multiple repositories.

REPOSITORY ALIASES

mgit alias add <alias> <directory> -> Create an alias for a repository directory.
mgit alias list                    -> List all repository aliases.
mgit unalias <alias>               -> Remove a repository alias.

Examples: mgit alias add backend C:\Projects\backend
mgit backend status
mgit backend pull
mgit backend checkout develop
mgit frontend,backend status
mgit all status

COMMAND SHORTCUTS

mgit shortcut add <name> <command...> -> Save a command as a named shortcut.
mgit shortcut run <name>              -> Open a new terminal and run the shortcut.
mgit shortcut list                    -> List all command shortcuts.
mgit shortcut remove <name>           -> Remove a command shortcut.

Examples:

mgit shortcut add frontend "cd C:\Projects\app\frontend && npm run dev"
mgit shortcut add backend "cd C:\Projects\app\backend && go run ."

mgit shortcut run frontend
mgit shortcut run backend

Shortcuts are executed by the operating system's shell. You can use any commands, arguments, pipes, redirects, etc.

OTHER

mgit help -> Show this help message.
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
	a, _ := json.MarshalIndent(aliases, "", "    ")
	_ = os.WriteFile(aliasConfigPath, a, 0644)
}

func saveShortcuts(shortcuts map[string]string) {
	a, _ := json.MarshalIndent(shortcuts, "", "    ")
	_ = os.WriteFile(shortcutConfigPath, a, 0644)
}

func main() {
	var wg sync.WaitGroup

	if err != nil {
		panic(err)
	}

	err = os.MkdirAll(filepath.Join(userConfigDir, "mgit"), 0755)
	if err != nil {
		panic(err)
	}

	_, err := os.OpenFile(aliasConfigPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	_, err = os.OpenFile(shortcutConfigPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	aliases := make(map[string]string)

	data, err := os.ReadFile(aliasConfigPath)
	if err != nil {
		panic(err)
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &aliases); err != nil {
			panic(err)
		}
	}

	shortcuts := make(map[string]string)

	data, err = os.ReadFile(shortcutConfigPath)
	if err != nil {
		panic(err)
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &shortcuts); err != nil {
			panic(err)
		}
	}

	// Index CWD
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

		if len(os.Args) < 3 {
			fmt.Println(msg)
			return
		}

		if os.Args[2] == "add" {
			if len(os.Args) != 5 {
				fmt.Println("\nUsage: mgit alias add <alias> <directory>\n")
				return
			}

			aliases[os.Args[3]] = resolveAbsolutePath(os.Args[4], aliases)
			saveAliases(aliases)

		} else if os.Args[2] == "list" {

			if len(aliases) == 0 {
				fmt.Println("\nNo aliases added.")
				return
			}

			fmt.Println("\nAliases:")
			fmt.Println("────────────────────────────────────────────")

			for alias, path := range aliases {
				fmt.Printf("%-20s %s\n", alias, path)
			}
		}

	} else if s == "unalias" {

		if len(os.Args) < 3 {
			fmt.Println("Usage: mgit unalias <alias>")
			return
		}

		delete(aliases, os.Args[2])
		saveAliases(aliases)

	} else if s == "shortcut" {

		if len(os.Args) < 3 {
			fmt.Println(msg)
			return
		}

		s1 := os.Args[2]

		if s1 == "add" {
			// Preserve spaces between command arguments.
			shortcuts[os.Args[3]] = strings.Join(os.Args[4:], " ")
			saveShortcuts(shortcuts)

		} else if s1 == "run" {

			if len(os.Args) != 4 {
				fmt.Println("Usage: mgit shortcut run <shortcut>")
				return
			}

			command, ok := shortcuts[os.Args[3]]
			if !ok {
				fmt.Printf("Shortcut %q does not exist.\n", os.Args[3])
				return
			}

			if err := openTerminal(command); err != nil {
				fmt.Println("Failed to open terminal:", err)
			}

		} else if s1 == "list" {

			if len(shortcuts) == 0 {
				fmt.Println("\nNo shortcuts added.")
				return
			}

			fmt.Println("\nShortcuts:")
			fmt.Println("────────────────────────────────────────────")

			for shortcut, command := range shortcuts {
				fmt.Printf("%-20s %s\n", shortcut, command)
			}

		} else if s1 == "remove" {

			if len(os.Args) != 4 {
				fmt.Println("Usage: mgit shortcut remove <shortcut>")
				return
			}

			delete(shortcuts, os.Args[3])
			saveShortcuts(shortcuts)

		} else {
			fmt.Println(msg)
			return
		}

	} else {

		// Treat actual git commands
		if s == "all" {

			for i := 0; i < len(dirs); i++ {
				var repo = dirs[i]

				wg.Add(1)
				go runGitCommand(repo, os.Args[2:], &wg)
			}

			wg.Wait()

		} else if strings.Contains(s, ",") {

			// Treat multiple repos at the same time
			var a []string = strings.Split(s, ",")

			for _, v := range a {
				wg.Add(1)

				go runGitCommand(
					resolveAbsolutePath(v, aliases),
					os.Args[2:],
					&wg,
				)
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
