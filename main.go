package main

import (
	"fmt"
	"iapodcast/summary"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env.local")

	providerNames := []string{"claude", "openai"}
	promptProviders := summary.AllPromptProviders()

	for _, providerName := range providerNames {
		for _, promptProvider := range promptProviders {
			provider := summary.NewSummaryProvider(providerName, promptProvider)

			err := provider.Prepare()
			if err != nil {
				fmt.Println(err)
				continue
			}

			summarizer := NewYouTubeSummarizer(provider)

			youtubeURL := "https://youtu.be/PebaNrEFWIs"
			fmt.Println(youtubeURL)
			if err := summarizer.ProcessVideo(youtubeURL); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}
