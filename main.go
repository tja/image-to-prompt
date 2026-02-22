package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Define root command
var RootCmd = &cobra.Command{
	Use:               "image-to-prompt [flags] <image-file>",
	Args:              cobra.ExactArgs(1),
	Version:           "0.0.1",
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	PersistentPreRunE: setup,
	RunE:              run,
}

func init() {
	// Add flags
	RootCmd.PersistentFlags().String("log-level", "warn", "verbosity of logging output")
	RootCmd.PersistentFlags().Bool("log-as-json", false, "change logging format to JSON")
}

// main is the entry point of the application.
func main() {
	if err := RootCmd.Execute(); err != nil {
		slog.Error("Failed to execute command", slog.Any("error", err))
		os.Exit(1)
	}
}

// setup sets up the application.
func setup(cmd *cobra.Command, _ []string) error {
	// Logging level and format
	logLevel, err := cmd.Flags().GetString("log-level")
	if err != nil {
		return fmt.Errorf("get log-level flag: %w", err)
	}

	logAsJSON, err := cmd.Flags().GetBool("log-as-json")
	if err != nil {
		return fmt.Errorf("get log-as-json flag: %w", err)
	}

	var level slog.Level

	err = level.UnmarshalText([]byte(logLevel))
	if err != nil {
		return fmt.Errorf("parse log level: %w", err)
	}

	var handler slog.Handler

	if logAsJSON {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}

	slog.SetDefault(slog.New(handler))

	return nil
}

// run loads an image and constructs a run-length encoded prompt describing the image pixel by pixel.
func run(_ *cobra.Command, args []string) error {
	// Open image file
	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("open image file: %w", err)
	}

	defer f.Close()

	// Decode image
	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	// Build prompt
	var prompt strings.Builder
	bounds := img.Bounds()

	fmt.Fprintf(&prompt, "Please create an image with %d rows and %d columns.\n\n", bounds.Dy(), bounds.Dx())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		row := y - bounds.Min.Y
		x := bounds.Min.X

		// Determine color of first pixel
		currentColor := blackOrWhite(img.At(x, y))
		runLength := 1
		x++

		// Walk remaining row
		for x < bounds.Max.X {
			c := blackOrWhite(img.At(x, y))
			if c == currentColor {
				runLength++
				x++

				continue
			}

			// Flush current run
			if runLength == 1 {
				if runLength == (x - bounds.Min.X) {
					fmt.Fprintf(&prompt, "Line %d starts with 1 %s pixel, ", row+1, currentColor)
				} else {
					fmt.Fprintf(&prompt, "followed by 1 %s pixel, ", currentColor)
				}
			} else {
				if runLength == (x - bounds.Min.X) {
					fmt.Fprintf(&prompt, "Line %d starts with %d %s pixels, ", row+1, runLength, currentColor)
				} else {
					fmt.Fprintf(&prompt, "followed by %d %s pixels, ", runLength, currentColor)
				}
			}

			currentColor = c
			runLength = 1
			x++
		}

		// Flush last run of row
		if runLength == bounds.Dx() {
			fmt.Fprintf(&prompt, "Line %d only contains %s pixels.\n", row+1, currentColor)
		} else if runLength == 1 {
			fmt.Fprintf(&prompt, "and finally 1 %s pixel.\n", currentColor)
		} else {
			fmt.Fprintf(&prompt, "and finally %d %s pixels.\n", runLength, currentColor)
		}
	}

	fmt.Print(prompt.String())

	return nil
}

// blackOrWhite returns "black" if the pixel's gray value is <50%, otherwise "white".
func blackOrWhite(c color.Color) string {
	gray := color.GrayModel.Convert(c).(color.Gray) //nolint:errcheck
	if gray.Y < 128 {
		return "black"
	}

	return "white"
}
