package main

import (
    "tfvarenv/cmd"
)

func main() {
    rootCmd := cmd.NewRootCmd()
    if err := rootCmd.Execute(); err != nil {
        panic(err)
    }
}
