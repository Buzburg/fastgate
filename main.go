package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/buzburgai-arch/fastgate/pkg/proxy"
	"github.com/buzburgai-arch/fastgate/pkg/triage"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "start":
		startCmd := flag.NewFlagSet("start", flag.ExitOnError)
		port := startCmd.Int("port", 8080, "Port to listen on")
		upstream := startCmd.String("upstream", "https://api.anthropic.com", "Upstream LLM API base URL")
		_ = startCmd.Parse(os.Args[2:])

		srv, err := proxy.NewServer(*upstream, *port)
		if err != nil {
			log.Fatalf("Error configuring server: %v", err)
		}
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}

	case "triage":
		if len(os.Args) < 3 {
			fmt.Println("Usage: fastgate triage <prompt>")
			os.Exit(1)
		}
		prompt := os.Args[2]
		classifier := triage.NewClassifier()
		availableTools := []string{
			"read_file", "view_file", "list_dir", "write_to_file",
			"replace_file_content", "run_command", "search_web",
			"invoke_subagent", "manage_task", "send_message",
		}

		res := classifier.Classify(prompt, availableTools)
		fmt.Printf("Prompt: %q\n", prompt)
		fmt.Printf("Decision Latency: %s\n", res.DecisionDuration)
		fmt.Printf("Retained Tools (%d/%d): %v\n", len(res.RetainedTools), res.OriginalToolCount, res.RetainedTools)
		fmt.Printf("Token Savings: %.1f%%\n", res.TokenSavingsPct)
		fmt.Printf("Verdict: %s\n", res.System1Verdict)

	case "version":
		fmt.Printf("fastgate v%s (System 1 Local Agent Gate)\n", version)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`fastgate — Sub-15ms Local System 1 Decision Proxy for LLM Agents

Commands:
  start     Run the local proxy daemon
            --port <int>        Port (default 8080)
            --upstream <url>    Upstream LLM URL (default: https://api.anthropic.com)

  triage    Run test triage on a sample user prompt
            Usage: fastgate triage "Read package.json and run tests"

  version   Show version information`)
}
