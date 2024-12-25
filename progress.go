package main

import (
	"fmt"

	"github.com/schollz/progressbar/v3"
)

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

