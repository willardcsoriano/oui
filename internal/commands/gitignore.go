package commands

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var stackPatterns = map[string][]string{
	"Python": {
		"__pycache__/", "*.py[cod]", "*.egg-info/", "dist/", "build/",
		".venv/", "venv/", ".env",
	},
	"Node": {
		"node_modules/", "dist/", "build/", ".env", ".env.local", "*.log",
	},
	"Go": {
		"*.exe", "*.test", "*.out", "/bin/", "/dist/",
	},
	"Rust": {
		"/target/", "Cargo.lock",
	},
	"Java": {
		"*.class", "*.jar", "*.war", "target/", ".gradle/", "build/",
	},
	"Ruby": {
		".bundle/", "vendor/bundle/", "*.gem",
	},
	"Dotnet": {
		"bin/", "obj/", "*.user", "*.suo",
	},
}

var stackDetectors = map[string][]string{
	"Python": {"pyproject.toml", "setup.py", "requirements.txt", "Pipfile"},
	"Node":   {"package.json", "yarn.lock", "pnpm-lock.yaml"},
	"Go":     {"go.mod", "go.sum"},
	"Rust":   {"Cargo.toml"},
	"Java":   {"pom.xml", "build.gradle", "build.gradle.kts"},
	"Ruby":   {"Gemfile", "Rakefile"},
	"Dotnet": {"*.csproj", "*.sln", "*.fsproj"},
}

func handleGitignore(_ string, w io.Writer) (bool, error) {
	stacks := detectStacks()

	existing := map[string]bool{}
	content, _ := os.ReadFile(".gitignore")
	for _, line := range strings.Split(string(content), "\n") {
		existing[strings.TrimSpace(line)] = true
	}

	var added []string
	var toAdd []string
	for _, stack := range stacks {
		patterns := stackPatterns[stack]
		for _, p := range patterns {
			if !existing[p] {
				toAdd = append(toAdd, p)
				added = append(added, p)
			}
		}
	}

	if len(toAdd) == 0 {
		fmt.Fprintln(w, ".gitignore is already up to date")
		return true, nil
	}

	f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return true, fmt.Errorf("gitignore: open: %w", err)
	}
	defer f.Close()

	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		fmt.Fprintln(f)
	}
	for _, p := range toAdd {
		fmt.Fprintln(f, p)
	}

	fmt.Fprintf(w, "added %d entries to .gitignore (%s)\n", len(added), strings.Join(stacks, ", "))
	return true, nil
}

func detectStacks() []string {
	var found []string
	for stack, markers := range stackDetectors {
		for _, m := range markers {
			if fileExists(m) {
				found = append(found, stack)
				break
			}
		}
	}
	return found
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
