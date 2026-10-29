package app

import "fmt"

func runUsage() {
	fmt.Println(`
publisher — Publish and manage articles from your local files

Usage:
	publisher <command> [arguments]

Commands:
	daemon             	Start the background file watcher
	publish <file>     	Publish a file
	policy <file>    	Check file post policy before publication
	version            	Show version information
	help               	Show this help message

Examples:
	publisher daemon
	publisher policy articles/my-article.md
	publisher publish articles/my-article.md
	publisher version

Run 'publisher help <command>' for more information about a specific command.
	`)
}
