package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Automata-Labs-team/code-sandbox-mcp/installer"
	deps "github.com/Automata-Labs-team/code-sandbox-mcp/languages"
	"github.com/Automata-Labs-team/code-sandbox-mcp/resources"
	"github.com/Automata-Labs-team/code-sandbox-mcp/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GenerateEnumTag generates the jsonschema enum tag for all supported languages
func GenerateEnumTag() string {
	var tags []string
	for _, lang := range deps.AllLanguages {
		tags = append(tags, fmt.Sprintf("enum=%s", lang))
	}
	return strings.Join(tags, ",")
}

func main() {
	// Check for --install flag
	installFlag := flag.Bool("install", false, "Add this binary to Claude Desktop config")
	noUpdateFlag := flag.Bool("no-update", false, "Disable auto-update check")
	flag.Parse()

	if *installFlag {
		if err := installer.InstallConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check for updates unless disabled
	if !*noUpdateFlag {
		if hasUpdate, downloadURL, err := installer.CheckForUpdate(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to check for updates: %v\n", err)
		} else if hasUpdate {
			fmt.Println("Updating to new version...")
			if err := installer.PerformUpdate(downloadURL); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to update: %v\n", err)
			}
			// No need for else block - performUpdate will either exit or return an error
		}
	}

	s := server.NewMCPServer("code-sandbox-mcp", "v1.0.0", server.WithLogging(), server.WithResourceCapabilities(true, true))

	// Register a tool to run code in a docker container
	runCodeTool := mcp.NewTool("run_code",
		mcp.WithDescription(
			"Run code in a docker container with automatic dependency detection and installation. \n"+
				"The tool will analyze your code and install required packages automatically. \n"+
				"The supported languages are: "+GenerateEnumTag()+". \n"+
				"Returns the execution logs of the container.",
		),
		mcp.WithString("code",
			mcp.Required(),
			mcp.Description("The code to run"),
		),
		mcp.WithString("language",
			mcp.Required(),
			mcp.Description("The programming language to use"),
			mcp.Enum(deps.AllLanguages.ToArray()...),
		),
	)

	runProjectTool := mcp.NewTool("run_project",
		mcp.WithDescription(
			"Run a code project in a docker container. \n"+
				"The tool will analyze your code and install required packages automatically. \n"+
				"The supported languages are: "+GenerateEnumTag()+". \n"+
				"Returns the initial logs of the container and resource URI to access the container logs Resource.",
		),
		mcp.WithString("projectDir",
			mcp.Required(),
			mcp.Description("Location of the project to run"),
		),
		mcp.WithString("language",
			mcp.Required(),
			mcp.Description("The programming language to use"),
			mcp.Enum(deps.AllLanguages.ToArray()...),
		),
		mcp.WithString("entrypointCmd",
			mcp.Required(),
			mcp.Description("Entrypoint command to run at the root of the project directory."),
		),
	)

	// Register dynamic resource for container logs
	// Dynamic resource example - Container Logs by ID
	template := mcp.NewResourceTemplate(
		"containers://{id}/logs",
		"Container Logs",
		mcp.WithTemplateDescription("Returns all container logs from the specified container. Logs are returned as a single text resource."),
		mcp.WithTemplateMIMEType("text/plain"),
		mcp.WithTemplateAnnotations([]mcp.Role{mcp.RoleAssistant, mcp.RoleUser}, 0.5),
	)

	s.AddResourceTemplate(template, resources.GetContainerLogs)
	s.AddTool(runCodeTool, tools.RunCodeSandbox)
	s.AddTool(runProjectTool, tools.RunProjectSandbox)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

}
