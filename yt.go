package main

import (
	"bufio"
	"fmt"
	"iapodcast/summary"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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

func (self *YouTubeSummarizer) Summarize(transcription string, videoId string, startTime time.Time) error {
	summary, err := self.generateSummary(transcription)
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
		audioPath, err := self.downloadAudio(url)
		if err != nil {
			return fmt.Errorf("download failed: %v", err)
		}
		defer os.Remove(audioPath)

		transcription, err = self.transcribeAudio(audioPath)
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

func (self *YouTubeSummarizer) downloadAudio(url string) (string, error) {
	self.progress.Update("Creating download directory...")
	if err := os.MkdirAll(self.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	outputTemplate := filepath.Join(self.outputDir, "%(id)s.%(ext)s")

	cmd := exec.Command("yt-dlp",
		"-N", "5",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "192K",
		"-o", outputTemplate,
		"--newline",
		url)

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

	return files[0], nil
}

// Not used for now, should we keep it?
func splitAudioIfNeeded(audioPath string, maxSizeBytes int64) ([]string, error) {
	fileInfo, err := os.Stat(audioPath)
	if err != nil {
		return nil, fmt.Errorf("cannot get file info: %v", err)
	}

	if fileInfo.Size() <= maxSizeBytes {
		return []string{audioPath}, nil
	}

	numParts := (fileInfo.Size() / maxSizeBytes) + 1

	cmd := exec.Command("ffmpeg", "-i", audioPath, "-f", "null", "-")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get duration: %v", err)
	}

	durationStr := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2})`).FindStringSubmatch(string(output))
	if len(durationStr) < 4 {
		return nil, fmt.Errorf("couldn't parse duration")
	}

	hours, _ := strconv.Atoi(durationStr[1])
	minutes, _ := strconv.Atoi(durationStr[2])
	seconds, _ := strconv.Atoi(durationStr[3])
	totalSeconds := float64(hours*3600 + minutes*60 + seconds)

	segmentDuration := totalSeconds / float64(numParts)

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
			for _, segment := range segments {
				os.Remove(segment)
			}
			return nil, fmt.Errorf("failed to create segment %d: %v", i+1, err)
		}

		segments = append(segments, segmentPath)
	}

	return segments, nil
}
