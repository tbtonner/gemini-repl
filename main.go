package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/glamour"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// AppState holds the persistent configuration for our session
type AppState struct {
	client           *genai.Client
	model            *genai.GenerativeModel
	chat             *genai.ChatSession
	currentModelName string
	lastCodeBlock    string
	renderer         *glamour.TermRenderer
}

func main() {
	ctx := context.Background()
	state := initAppState(ctx)
	defer state.client.Close()

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\033[35m✨ Gemini REPL Ready (%s)\033[0m\n", state.currentModelName)

	for {
		fmt.Print("\033[36m> \033[0m")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}
		if handleCommands(input, state) {
			continue
		}
		if input == "exit" || input == "quit" {
			break
		}

		processChat(ctx, input, state)
	}
}

// initAppState handles the initial setup of the API client and rendering engine
func initAppState(ctx context.Context) *AppState {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Error: GEMINI_API_KEY not set.")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal(err)
	}

	// Use the latest 2026 standard model
	modelName := "gemini-3-flash-preview"
	model := client.GenerativeModel(modelName)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text("You are a concise, senior go developer. Give direct answers. Use Markdown for all code.")},
	}

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("notty"),
		glamour.WithWordWrap(100),
	)

	return &AppState{
		client:           client,
		model:            model,
		chat:             model.StartChat(),
		currentModelName: modelName,
		renderer:         renderer,
	}
}

// handleCommands checks for local REPL commands before sending text to the AI
func handleCommands(input string, state *AppState) bool {
	switch {
	case input == "clear":
		fmt.Print("\033[H\033[2J")
		state.chat = state.model.StartChat()
		fmt.Println("\033[35m✨ History cleared.\033[0m")
		return true

	case input == "copy":
		if state.lastCodeBlock != "" {
			clipboard.WriteAll(state.lastCodeBlock)
			fmt.Println("\033[32m📋 Copied last code block to clipboard!\033[0m")
		} else {
			fmt.Println("No code found in the last response.")
		}
		return true

	case strings.HasPrefix(input, "model "):
		state.currentModelName = strings.TrimPrefix(input, "model ")
		state.model = state.client.GenerativeModel(state.currentModelName)
		state.chat = state.model.StartChat()
		fmt.Printf("\033[35m🔄 Switched to %s\033[0m\n", state.currentModelName)
		return true
	}
	return false
}

// processChat manages the loading animation, the API call, and the output rendering
func processChat(ctx context.Context, input string, state *AppState) {
	stopLoading := make(chan bool)
	go showLoadingIndicator(state.currentModelName, stopLoading)

	iter := state.chat.SendMessageStream(ctx, genai.Text(input))
	var fullResponse strings.Builder
	firstChunk := true

	for {
		resp, err := iter.Next()
		if firstChunk {
			stopLoading <- true
			firstChunk = false
		}
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			break
		}

		for _, cand := range resp.Candidates {
			for _, part := range cand.Content.Parts {
				fullResponse.WriteString(fmt.Sprint(part))
			}
		}
	}

	// Extract code block for potential copying later
	extractLastCode(fullResponse.String(), state)

	// Render the final Markdown to the terminal
	out, _ := state.renderer.Render(fullResponse.String())
	fmt.Print(out)
}

func showLoadingIndicator(modelName string, stop chan bool) {
	dots := []string{".  ", ".. ", "..."}
	i := 0
	for {
		select {
		case <-stop:
			fmt.Print("\r                           \r") // Clear the line
			return
		default:
			fmt.Printf("\r\033[90m%s is thinking%s\033[0m", modelName, dots[i%3])
			i++
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func extractLastCode(text string, state *AppState) {
	re := regexp.MustCompile("(?s)```(?:[a-z]+)?\n(.*?)\n```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		state.lastCodeBlock = matches[1]
	}
}
