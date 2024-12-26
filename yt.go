package main

import (
	"bufio"
	"fmt"
	"iapodcast/ai"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type YouTubeSummarizer struct {
	provider  ai.AiServiceProvider
	outputDir string
	progress  *Progress
}

func NewYouTubeSummarizer(provider ai.AiServiceProvider) *YouTubeSummarizer {
	return &YouTubeSummarizer{
		provider:  provider,
		outputDir: "downloads",
		progress:  NewProgress(),
	}
}

func (self *YouTubeSummarizer) Summarize(transcription string, videoId string, startTime time.Time) error {
	summary, err := self.provider.GenerateFromInput(transcription)
	if err != nil {
		return fmt.Errorf("summarization failed: %v", err)
	}

	elapsed := time.Since(startTime)

	fmt.Printf("\nProcessing completed in %s\n", elapsed.Round(time.Second))

	summaryPath := path.Join(self.outputDir, fmt.Sprintf("%s-%s.txt", videoId, self.provider.String()))
	fmt.Println(summaryPath)
	err = os.WriteFile(summaryPath, []byte(summary), 0777)

	return err
}

func (self *YouTubeSummarizer) ProcessVideo(url string) error {
	defer self.progress.Clear()

	startTime := time.Now()
	self.progress.Update("Starting video processing...")

	var transcription string

	videoId, err := self.GetVideoIdFromUrl(url)
	if err != nil {
		return err
	}

	existingTranscriptionPath := path.Join(self.outputDir, videoId) + ".txt"
	_, err = os.Stat(existingTranscriptionPath)	

	if os.IsNotExist(err) {
		audioPath, err := self.downloadAudio(videoId, url)
		if err != nil {
			return fmt.Errorf("download failed: %v", err)
		}

		transcription, err = self.transcribeAudio(videoId, audioPath)
		if err != nil {
			return fmt.Errorf("transcription failed: %v", err)
		}
	} else {
		contentBytes, err := os.ReadFile(existingTranscriptionPath)
		if err != nil {
			return err
		}
		transcription = string(contentBytes)
	}

	err = self.Summarize(transcription, videoId, startTime)
	return err
}

func (self *YouTubeSummarizer) GetVideoIdFromUrl(url string) (string, error) {
    patterns := []*regexp.Regexp{
        regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/|youtube\.com/embed/)([^?&]+)`),
        regexp.MustCompile(`^([^?&]+)$`),
    }

    url = strings.TrimSpace(url)
    
    for _, pattern := range patterns {
        if matches := pattern.FindStringSubmatch(url); len(matches) > 1 {
            return matches[1], nil
        }
    }

    return "", fmt.Errorf("invalid YouTube URL or video ID: %s", url)
}

func (self *YouTubeSummarizer) downloadAudio(videoId string, url string) (string, error) {
	self.progress.Update("Creating download directory...")
	if err := os.MkdirAll(self.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	outputTemplate := filepath.Join(self.outputDir, "%(id)s.%(ext)s")

	cmd := exec.Command(
		"yt-dlp",
		"-N", "5",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "192K",
		"-o", outputTemplate,
		"--newline",
		url,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start download: %v", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "%") {
				self.progress.Update(fmt.Sprintf("Downloading: %s", line))
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to download audio: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(self.outputDir, "*.mp3"))
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no output file found")
	}

	filePath := fmt.Sprintf("downloads/%s.wav", videoId)
	cmd = exec.Command(
		"ffmpeg",
		"-i", files[0],
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		filePath,
	)
	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to convert: %v", err)
	}

	return filePath, nil
}

func (self *YouTubeSummarizer) transcribeAudio(videoId string, audioPath string) (string, error) {
	outputFile := audioPath + ".txt"
	outFile, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	cmd := exec.Command(
		"./whishper-cli",
		"-np",
		"-t", "10",
		"-l", "auto",
		"-m", "whisper.cpp/models/ggml-base.bin",
		"-of", fmt.Sprintf("downloads/%s", videoId),
		"-otxt", "true",
		"-f", audioPath,
	)
	cmd.Stdout = outFile

	self.progress.Update("Transcribing audio...")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to transcribe: %v", err)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription file: %v", err)
	}

	return string(content), nil
}

