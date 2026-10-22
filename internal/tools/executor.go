package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/willardcsoriano/oui/internal/providers"
)

var Schemas = []providers.ToolSchema{
	{
		Name:        "web_search",
		Description: "Search the web using DuckDuckGo and return a summary plus relevant links. Use this for current information, documentation lookups, or anything requiring up-to-date knowledge.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Search query.",
				},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "fetch_url",
		Description: "Fetch the content of a URL and return it as plain text. Useful for reading documentation, API references, or any web page. HTML is automatically converted to readable text.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "Full URL to fetch (must start with http:// or https://).",
				},
			},
			"required": []string{"url"},
		},
	},
	{
		Name:        "read_file",
		Description: "Read the contents of a file at the given path.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Relative or absolute file path to read.",
				},
			},
			"required": []string{"path"},
		},
	},
	{
		Name:        "write_file",
		Description: "Write content to a file, creating it or overwriting it.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path to write.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file.",
				},
			},
			"required": []string{"path", "content"},
		},
	},
	{
		Name:        "bash",
		Description: "Execute a shell command and return combined stdout and stderr.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Shell command to run.",
				},
			},
			"required": []string{"command"},
		},
	},
}

// denyPatterns are unconditional blocks — not user-configurable.
var denyPatterns = []string{
	"git push --force",
	"git push -f",
	"git reset --hard",
	"git clean -f",
	"rm -rf",
	"rm -fr",
	":(){ :|:& };:", // fork bomb
	"dd if=",
	"mkfs",
	"> /dev/",
}

func checkDenyList(command string) error {
	lower := strings.ToLower(command)
	for _, pat := range denyPatterns {
		if strings.Contains(lower, pat) {
			return fmt.Errorf("command blocked by deny list: %q matches pattern %q", command, pat)
		}
	}
	return nil
}

// Execute dispatches a tool call by name with JSON-encoded args.
func Execute(name, argsJSON string) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("malformed tool args: %w", err)
	}

	switch name {
	case "web_search":
		return webSearch(args)
	case "fetch_url":
		return fetchURL(args)
	case "read_file":
		return readFile(args)
	case "write_file":
		return writeFile(args)
	case "bash":
		return bash(args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func readFile(args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("read_file: path must be a non-empty string")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read_file: %w", err)
	}
	return string(data), nil
}

func writeFile(args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("write_file: path must be a non-empty string")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("write_file: content must be a string")
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write_file: %w", err)
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path), nil
}

func bash(args map[string]any) (string, error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("bash: command must be a non-empty string")
	}
	if err := checkDenyList(command); err != nil {
		return "", err
	}
	cmd := exec.Command("bash", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Return output even on non-zero exit — the model needs to see stderr.
		return string(out), nil
	}
	return string(out), nil
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// webSearch queries DuckDuckGo Instant Answers and returns a formatted summary.
// For full web results, combine with fetch_url on the returned links.
func webSearch(args map[string]any) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("web_search: query must be a non-empty string")
	}

	apiURL := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json&no_html=1&skip_disambig=1"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("web_search: %w", err)
	}
	req.Header.Set("User-Agent", "oui/dev")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("web_search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", fmt.Errorf("web_search: reading response: %w", err)
	}

	var result struct {
		Heading     string `json:"Heading"`
		Abstract    string `json:"AbstractText"`
		AbstractURL string `json:"AbstractURL"`
		Answer      string `json:"Answer"`
		Definition  string `json:"Definition"`
		DefinitionURL string `json:"DefinitionURL"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
			Topics   []struct {
				Text     string `json:"Text"`
				FirstURL string `json:"FirstURL"`
			} `json:"Topics"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("web_search: parsing response: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Search: " + query + "\n\n")
	if result.Answer != "" {
		sb.WriteString("Answer: " + result.Answer + "\n\n")
	}
	if result.Abstract != "" {
		sb.WriteString(result.Abstract)
		if result.AbstractURL != "" {
			sb.WriteString("\nSource: " + result.AbstractURL)
		}
		sb.WriteString("\n\n")
	}
	if result.Definition != "" {
		sb.WriteString("Definition: " + result.Definition)
		if result.DefinitionURL != "" {
			sb.WriteString("\nSource: " + result.DefinitionURL)
		}
		sb.WriteString("\n\n")
	}

	count := 0
	addTopic := func(text, link string) {
		if count >= 8 || text == "" || link == "" {
			return
		}
		sb.WriteString("- " + text + "\n  " + link + "\n")
		count++
	}
	for _, t := range result.RelatedTopics {
		addTopic(t.Text, t.FirstURL)
		for _, sub := range t.Topics {
			addTopic(sub.Text, sub.FirstURL)
		}
	}
	for _, r := range result.Results {
		addTopic(r.Text, r.FirstURL)
	}

	if sb.Len() <= len("Search: "+query+"\n\n") {
		return "No results found for: " + query + "\nTry fetch_url with a specific documentation URL.", nil
	}
	return sb.String(), nil
}

// fetchURL retrieves a URL and returns its content as plain text.
// HTML pages are stripped of tags; other content types are returned as-is.
func fetchURL(args map[string]any) (string, error) {
	rawURL, ok := args["url"].(string)
	if !ok || rawURL == "" {
		return "", fmt.Errorf("fetch_url: url must be a non-empty string")
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return "", fmt.Errorf("fetch_url: url must start with http:// or https://")
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("fetch_url: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; oui/dev)")
	req.Header.Set("Accept", "text/html,text/plain,*/*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch_url: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB cap
	if err != nil {
		return "", fmt.Errorf("fetch_url: reading body: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		return htmlToText(body), nil
	}
	text := string(body)
	if len(text) > 50000 {
		text = text[:50000] + "\n...[truncated]"
	}
	return text, nil
}

// htmlToText extracts readable text from an HTML document.
func htmlToText(data []byte) string {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return string(data)
	}

	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "head", "nav", "footer", "aside", "form":
				return
			case "p", "div", "section", "article", "h1", "h2", "h3", "h4", "h5", "h6":
				sb.WriteString("\n")
			case "li":
				sb.WriteString("\n- ")
			case "br":
				sb.WriteString("\n")
			case "tr":
				sb.WriteString("\n")
			}
		}
		if n.Type == html.TextNode {
			if t := strings.TrimSpace(n.Data); t != "" {
				sb.WriteString(t + " ")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Collapse blank lines and trim.
	lines := strings.Split(sb.String(), "\n")
	var out []string
	prev := ""
	for _, l := range lines {
		l = strings.TrimRight(l, " ")
		if l == "" && prev == "" {
			continue
		}
		out = append(out, l)
		prev = l
	}
	text := strings.Join(out, "\n")
	if len(text) > 50000 {
		text = text[:50000] + "\n...[truncated]"
	}
	return text
}
