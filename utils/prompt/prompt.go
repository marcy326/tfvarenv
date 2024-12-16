package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PromptYesNo prompts the user for a yes/no response.
func PromptYesNo(prompt string, defaultValue bool) bool {
	defaultStr := "Y/n"
	if !defaultValue {
		defaultStr = "y/N"
	}
	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultValue
	}
	return input == "y" || input == "yes"
}
