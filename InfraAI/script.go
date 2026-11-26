package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const configFile = "conf.json"
const defaultSystemPrompt = "You are a helpful, concise assistant."

type Config struct {
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

// Return Base url "http://192.x.x.x:11434"
func (c Config) BaseURL() string {
	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(c.Ip, c.Port),
	}
	return u.String()
}

func verifyConnection(ip, port string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), timeout)
	if err != nil {
		return err
	}
	return conn.Close()
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	text, err := reader.ReadString('\n')
	if err != nil && text == "" {
		log.Fatal("Input closed.")
	}
	return strings.TrimSpace(text)
}

func setup(reader *bufio.Reader) (Config, error) {
	for {
		config, err := loadConfig()
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("%s is invalid (%v), recreating it.\n", configFile, err)
			}
			// Create conf.json if it doesn't exist
			config, err = createConfig(reader)
			if err != nil {
				return config, err
			}
		}

		if err := verifyConnection(config.Ip, config.Port, 3*time.Second); err == nil {
			return config, nil
		} else {
			fmt.Printf("Cannot reach %s:%s (%v), please re-enter.\n", config.Ip, config.Port, err)
			os.Remove(configFile)
		}
	}
}

func loadConfig() (Config, error) {
	var config Config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	if config.Ip == "" || config.Port == "" {
		return config, fmt.Errorf("missing ip or port")
	}
	return config, nil
}

func createConfig(reader *bufio.Reader) (Config, error) {
	var ip, port string
	for ip == "" || port == "" {
		ip = prompt(reader, "Enter Ollama device's IP address: ")
		port = prompt(reader, "Enter device's port (Ollama default is 11434): ")
	}
	config := Config{Ip: ip, Port: port}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return config, fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return config, fmt.Errorf("writing %s: %w", configFile, err)
	}
	return config, nil
}

type ModelsResponse struct {
	Data []ModelInfo `json:"data"`
}

type ModelInfo struct {
	ID string `json:"id"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type Transcript struct {
	Model    string    `json:"model"`
	SavedAt  time.Time `json:"saved_at"`
	Messages []Message `json:"messages"`
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	config, err := setup(reader)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connection successful")

	chat(reader, config.BaseURL()+"/v1")
}

func chat(reader *bufio.Reader, baseURL string) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 120 * time.Second
	client := &http.Client{Transport: transport}

	// Fetch Models
	models, err := fetchModels(client, baseURL)
	if err != nil {
		log.Fatalf("Failed to fetch models (is Ollama running?): %v", err)
	}
	if len(models) == 0 {
		log.Fatal("No models found. Run 'ollama pull <model>' on the server first.")
	}

	// Select Model
	selectedModel := selectModel(reader, models)
	systemPrompt := defaultSystemPrompt

	// Initialize Conversation History
	history := []Message{{Role: "system", Content: systemPrompt}}

	fmt.Printf("\nChatting with %s\n", selectedModel)
	printHelp()
	fmt.Println(strings.Repeat("-", 50))

	// Start Interactive Chat
	for {
		fmt.Print("\nYou: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading input: %v", err)
		}

		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			continue
		}

		// Intercept slash commands
		if strings.HasPrefix(trimmed, "/") {
			parts := strings.SplitN(trimmed, " ", 2)
			cmd := strings.ToLower(parts[0])
			arg := ""
			if len(parts) > 1 {
				arg = strings.TrimSpace(parts[1])
			}

			switch cmd {
			case "/help":
				printHelp()

			case "/clear":
				history = []Message{{Role: "system", Content: systemPrompt}}
				fmt.Println("Conversation history cleared.")

			case "/model":
				// Re-fetch models in case new ones were pulled while running
				if updated, err := fetchModels(client, baseURL); err == nil && len(updated) > 0 {
					models = updated
				}
				selectedModel = selectModel(reader, models)
				fmt.Printf("Switched model to %s (history preserved).\n", selectedModel)

			case "/system":
				if arg == "" {
					fmt.Printf("Current system prompt: %q\nUsage: /system <new prompt>\n", systemPrompt)
				} else {
					systemPrompt = arg
					history[0] = Message{Role: "system", Content: systemPrompt}
					fmt.Println("System prompt updated.")
				}

			case "/save":
				if len(history) <= 1 {
					fmt.Println("Nothing to save yet.")
					break
				}
				path, err := saveTranscript(arg, selectedModel, history)
				if err != nil {
					fmt.Printf("Save failed: %v\n", err)
				} else {
					fmt.Printf("Transcript saved to %s\n", path)
				}

			case "/exit", "/quit":
				fmt.Println("Cya Cutie Pie!")
				return

			default:
				fmt.Printf("Unknown command %q. Type /help to see available commands.\n", cmd)
			}
			continue
		}

		// Append user message to history
		history = append(history, Message{Role: "user", Content: trimmed})

		// Send full history to maintain context
		fmt.Print("\nAssistant: ")
		reply, err := streamChat(client, baseURL, selectedModel, history)
		fmt.Println()
		if err != nil {
			fmt.Printf("Error: %v\n", err)

			// Remove the failed user message so history stays valid
			history = history[:len(history)-1]
			continue
		}

		// Append assistant reply to history
		history = append(history, Message{Role: "assistant", Content: reply})
	}
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /help             Show this menu")
	fmt.Println("  /clear            Reset conversation history")
	fmt.Println("  /model            Switch to a different model")
	fmt.Println("  /system <prompt>  View or change the system prompt")
	fmt.Println("  /save [file]      Save transcript (.md default, or .json)")
	fmt.Println("  /exit             Quit the program")
}

func fetchModels(client *http.Client, baseURL string) ([]ModelInfo, error) {
	resp, err := client.Get(baseURL + "/models")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var result ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func selectModel(reader *bufio.Reader, models []ModelInfo) string {
	fmt.Println("\nAvailable models:")
	for i, m := range models {
		fmt.Printf("[%d] %s\n", i+1, m.ID)
	}

	for {
		choice, err := strconv.Atoi(prompt(reader, "\nSelect a model number: "))
		if err == nil && choice >= 1 && choice <= len(models) {
			return models[choice-1].ID
		}
		fmt.Printf("Please enter a number between 1 and %d.\n", len(models))
	}
}

func streamChat(client *http.Client, baseURL, model string, messages []Message) (string, error) {
	payload, err := json.Marshal(ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		return "", err
	}

	resp, err := client.Post(baseURL+"/chat/completions", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			token := chunk.Choices[0].Delta.Content
			fmt.Print(token)
			full.WriteString(token)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream interrupted: %w", err)
	}
	if full.Len() == 0 {
		return "", fmt.Errorf("empty response from model")
	}
	return full.String(), nil
}

func saveTranscript(path, model string, history []Message) (string, error) {
	if path == "" {
		path = "chat-" + time.Now().Format("20060102-150405") + ".md"
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		path += ".md"
		ext = ".md"
	}

	var data []byte
	switch ext {
	case ".json":
		var err error
		data, err = json.MarshalIndent(Transcript{
			Model:    model,
			SavedAt:  time.Now(),
			Messages: history,
		}, "", "  ")
		if err != nil {
			return "", err
		}
	case ".md":
		data = []byte(renderMarkdown(model, history))
	default:
		return "", fmt.Errorf("unsupported file type %q (use .md or .json)", ext)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

func renderMarkdown(model string, history []Message) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Chat transcript\n\n")
	fmt.Fprintf(&b, "- **Model:** %s\n", model)
	fmt.Fprintf(&b, "- **Saved:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	for _, m := range history {
		switch m.Role {
		case "system":
			fmt.Fprintf(&b, "> **System prompt:** %s\n\n", m.Content)
		case "user":
			fmt.Fprintf(&b, "## You\n\n%s\n\n", m.Content)
		case "assistant":
			fmt.Fprintf(&b, "## Assistant\n\n%s\n\n", m.Content)
		}
	}
	return b.String()
}

