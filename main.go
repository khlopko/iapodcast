package main

import (
	"fmt"
	"iapodcast/ai"
	"iapodcast/validate"
	"path"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env.local")

	youtubeURL := "https://youtu.be/-jtuH1H8dfI"

	providerTypes := []ai.AiServiceType{ai.OpenAiServiceType, ai.AnthropicServiceType}
	promptProviders := ai.AllPromptProviders()

	suffixes := []string{}
	for _, providerName := range providerTypes {
		for _, provider := range promptProviders {
			suffixes = append(suffixes, fmt.Sprintf("%s-%s", providerName, provider.String()))
		}
	}

	if false {
		evaluationProvider := ai.NewAiServiceProvider(ai.OpenAiServiceType, &validate.ExternalPromptProvider{})
		evaluationProvider.Prepare()

		summarizer := NewYouTubeSummarizer(evaluationProvider)
		videoId, _ := summarizer.GetVideoIdFromUrl(youtubeURL)

		evaluation := validate.TranscriptionEvaluation{
			Provider: evaluationProvider,
		}

		for _, suffix := range suffixes {
			input := validate.TranscriptionSummaryEvaluationInput{
				TranscriptionPath: path.Join("downloads", fmt.Sprintf("%s.txt", videoId)),
				SummaryPath: path.Join("downloads", fmt.Sprintf("%s-%s.txt", videoId, suffix)),
			}
			score, err := evaluation.CompletenessOfSummary(input)
			fmt.Printf("Evaluating \"%s\" with result: %f %+v\n\n", suffix, score, err)
			time.Sleep(10 * time.Second)
		}
	} else {
		for _, providerType := range providerTypes {
			for _, promptProvider := range promptProviders {
				provider := ai.NewAiServiceProvider(providerType, promptProvider)

				err := provider.Prepare()
				if err != nil {
					fmt.Println(err)
					continue
				}

				summarizer := NewYouTubeSummarizer(provider)

				fmt.Println(youtubeURL)
				if err := summarizer.ProcessVideo(youtubeURL); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}
		}
	}
}
