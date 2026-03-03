package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/glamour"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"gemini-repl/themes"
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

// Global flag to prevent signal handler from firing during initial startup
var isReady bool

func main() {
	ctx := context.Background()
	state := initAppState(ctx)
	defer state.client.Close()

	// Pass state to signal handling so it can re-print the specific prompt
	setupSignalHandling(state)

	if len(os.Args) > 1 {
		input := strings.Join(os.Args[1:], " ")
		processChat(ctx, input, state)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\033[35m✨ Gemini REPL Ready (%s)\033[0m\n", state.currentModelName)

	// Mark app as ready for signal-based redraws
	isReady = true

	for {
		printPrompt()

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if handleCommands(input, state) {
			continue
		}

		processChat(ctx, input, state)
	}
}

// printPrompt centralizes the visual style of the input line
func printPrompt() {
	fmt.Print("\033[36m> \033[0m")
}

// setupSignalHandling ensures that when the user brings the process back to the foreground with "fg", the prompt is
// reprinted correctly without requiring an Enter keypress.
func setupSignalHandling(state *AppState) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGCONT)

	go func() {
		for {
			<-sigs
			// Only trigger if the main loop has finished its initial setup
			if isReady {
				fmt.Print("\n") // Clear any shell artifacts like "[1] + continued"
				printPrompt()
			}
		}
	}()
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

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(themes.Kanagawa),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		log.Fatal(err)
	}

	var (
		modelName = "gemini-3-flash-preview"
		model     = client.GenerativeModel(modelName)
	)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text("You are a concise, senior go developer. Give direct answers. Use Markdown for all code.")},
	}

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
	case input == "help":
		fmt.Println("\033[33mAvailable commands:\033[0m")
		fmt.Println("\033[36m  clear\033[0m - Clear the conversation history")
		fmt.Println("\033[36m  copy\033[0m  - Copy the last code block to clipboard")
		fmt.Println("\033[36m  model [name]\033[0m - Switch to a different model")
		fmt.Println("\033[36m  exit, quit\033[0m - Exit the REPL")
		return true

	case input == "exit", input == "quit":
		fmt.Println("\033[35m👋 Goodbye!\033[0m")
		os.Exit(0)

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

	extractLastCode(fullResponse.String(), state)

	out, _ := state.renderer.Render(fullResponse.String())
	fmt.Print(out)
}

// showLoadingIndicator displays a simple animated "thinking" message while waiting for the API response
func showLoadingIndicator(modelName string, stop chan bool) {
	dots := []string{".  ", ".. ", "..."}
	i := 0
	for {
		select {
		case <-stop:
			fmt.Print("\r                           \r")
			return
		default:
			fmt.Printf("\r\033[90m%s is thinking%s\033[0m", modelName, dots[i%3])
			i++
			time.Sleep(300 * time.Millisecond)
		}
	}
}

// extractLastCode uses a regex to find the last Markdown code block in the response and saves it to state for clipboard
// copying
func extractLastCode(text string, state *AppState) {
	re := regexp.MustCompile("(?s)```(?:[a-z]+)?\n(.*?)\n```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		state.lastCodeBlock = matches[1]
	}
}
