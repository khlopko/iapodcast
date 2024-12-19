package main

import (
	"bufio"
	"fmt"
	"iapodcast/summary"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
)

type YouTubeSummarizer struct {
	provider  summary.SummaryProvider
	outputDir string
	progress  *Progress
}

type Progress struct {
	bar     *progressbar.ProgressBar
	current string
}

func NewProgress() *Progress {
	return &Progress{
		bar: progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("Starting..."),
			progressbar.OptionSetWidth(50),
			progressbar.OptionShowBytes(false),
			progressbar.OptionSetRenderBlankState(true),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		),
	}
}

func (p *Progress) Update(status string) {
	p.current = status
	p.bar.Describe(fmt.Sprintf("[cyan]%s[reset]", status))
	p.bar.Add(1)
}

func (p *Progress) Clear() {
	p.bar.Clear()
}

func NewYouTubeSummarizer(provider summary.SummaryProvider) *YouTubeSummarizer {
	return &YouTubeSummarizer{
		provider:  provider,
		outputDir: "downloads",
		progress:  NewProgress(),
	}
}

func (ys *YouTubeSummarizer) downloadAudio(url string) (string, error) {
	ys.progress.Update("Creating download directory...")
	if err := os.MkdirAll(ys.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	outputTemplate := filepath.Join(ys.outputDir, "%(id)s.%(ext)s")

	ys.progress.Update("Starting download with yt-dlp...")
	cmd := exec.Command("yt-dlp",
		"-N", "5",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "192K",
		"-o", outputTemplate,
		"--newline", // Important for progress tracking
		url)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start download: %v", err)
	}

	// Read stderr for progress updates
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "%") {
				ys.progress.Update(fmt.Sprintf("Downloading: %s", line))
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to download audio: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(ys.outputDir, "*.mp3"))
	if err != nil || len(files) == 0 {
		return "", fmt.Errorf("no output file found")
	}

	return files[0], nil
}

func splitAudioIfNeeded(audioPath string, maxSizeBytes int64) ([]string, error) {
	// Check file size
	fileInfo, err := os.Stat(audioPath)
	if err != nil {
		return nil, fmt.Errorf("cannot get file info: %v", err)
	}

	// If file is under limit, return original
	if fileInfo.Size() <= maxSizeBytes {
		return []string{audioPath}, nil
	}

	// Calculate duration to split file
	numParts := (fileInfo.Size() / maxSizeBytes) + 1

	// Get total duration using FFmpeg
	cmd := exec.Command("ffmpeg", "-i", audioPath, "-f", "null", "-")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get duration: %v", err)
	}

	// Parse duration from FFmpeg output
	durationStr := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2})`).FindStringSubmatch(string(output))
	if len(durationStr) < 4 {
		return nil, fmt.Errorf("couldn't parse duration")
	}

	hours, _ := strconv.Atoi(durationStr[1])
	minutes, _ := strconv.Atoi(durationStr[2])
	seconds, _ := strconv.Atoi(durationStr[3])
	totalSeconds := float64(hours*3600 + minutes*60 + seconds)

	// Calculate segment duration
	segmentDuration := totalSeconds / float64(numParts)

	// Create segments
	var segments []string
	dir := filepath.Dir(audioPath)
	baseName := filepath.Base(audioPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := baseName[:len(baseName)-len(ext)]

	for i := 0; i < int(numParts); i++ {
		startTime := float64(i) * segmentDuration
		segmentPath := filepath.Join(dir, fmt.Sprintf("%s_part%d%s", nameWithoutExt, i+1, ext))

		cmd := exec.Command("ffmpeg",
			"-i", audioPath,
			"-ss", fmt.Sprintf("%.2f", startTime),
			"-t", fmt.Sprintf("%.2f", segmentDuration),
			"-c", "copy",
			segmentPath)

		if err := cmd.Run(); err != nil {
			// Cleanup any created segments on error
			for _, segment := range segments {
				os.Remove(segment)
			}
			return nil, fmt.Errorf("failed to create segment %d: %v", i+1, err)
		}

		segments = append(segments, segmentPath)
	}

	return segments, nil
}

func (ys *YouTubeSummarizer) transcribeAudio(audioPath string) (string, error) {
	// Create output file
	outputFile := audioPath + ".txt"
	outFile, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Run whisper with Python
	cmd := exec.Command("python3.10", "-m", "whisper", "--model", "base", "--output_dir", "downloads", "--output_format", "txt", audioPath)
	cmd.Stdout = outFile

	ys.progress.Update("Transcribing audio...")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to transcribe: %v", err)
	}

	// Read the transcription
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription file: %v", err)
	}

	// Clean up
	os.Remove(outputFile)

	return string(content), nil
}

func (ys *YouTubeSummarizer) generateSummary(transcription string) (string, error) {
	ys.progress.Update("Preparing summary request...")
	return ys.provider.GenerateFromInput(transcription)
}

func (ys *YouTubeSummarizer) Summarize(transcription string, startTime time.Time) error {
	summary, err := ys.generateSummary(transcription)
	if err != nil {
		return fmt.Errorf("summarization failed: %v", err)
	}

	elapsed := time.Since(startTime)

	fmt.Printf("\nProcessing completed in %s\n", elapsed.Round(time.Second))

	fmt.Println("\nSummary:")
	fmt.Println(summary)

	return nil
}

func (ys *YouTubeSummarizer) ProcessVideo(url string) error {
	defer ys.progress.Clear()

	startTime := time.Now()
	ys.progress.Update("Starting video processing...")

	audioPath, err := ys.downloadAudio(url)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	//defer os.Remove(audioPath)

	transcription, err := ys.transcribeAudio(audioPath)
	if err != nil {
		return fmt.Errorf("transcription failed: %v", err)
	}

	err = ys.Summarize(transcription, startTime)

	return err
}

func main() {
	godotenv.Load(".env.local")

	provider := summary.NewSummaryProvider("claude")

	err := provider.Prepare()
	if err != nil {
		fmt.Println(err)
		return
	}

	summarizer := NewYouTubeSummarizer(provider)

	youtubeURL := "https://youtu.be/PebaNrEFWIs"
	fmt.Println(youtubeURL)
	if err := summarizer.ProcessVideo(youtubeURL); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	/*
	bytes, err := os.ReadFile("downloads/DRgI8OEzqYY.txt")
	if err != nil {
		fmt.Println(err)
		return
	}

	transcription := string(bytes)
	summarizer.Summarize(transcription, time.Now())
	*/
}
