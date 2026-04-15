package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

var imageAspectRatioSeparators = strings.NewReplacer(
	"：", ":",
	"／", ":",
	"/", ":",
	"×", ":",
	"*", ":",
	"x", ":",
	"X", ":",
)

var canonicalImageAspectRatioLabels = map[string]string{
	"7:3":  "21:9",
	"16:9": "16:9",
	"4:3":  "4:3",
	"3:2":  "3:2",
	"1:1":  "1:1",
	"9:16": "9:16",
	"3:4":  "3:4",
	"2:3":  "2:3",
	"5:4":  "5:4",
	"4:5":  "4:5",
}

func normalizePortableAspectRatio(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	normalized := imageAspectRatioSeparators.Replace(value)
	parts := strings.Split(normalized, ":")
	if len(parts) != 2 {
		return value
	}

	width, errWidth := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errHeight := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errWidth != nil || errHeight != nil || width <= 0 || height <= 0 {
		return value
	}

	g := gcd(width, height)
	if g <= 0 {
		return fmt.Sprintf("%d:%d", width, height)
	}
	key := fmt.Sprintf("%d:%d", width/g, height/g)
	if label, ok := canonicalImageAspectRatioLabels[key]; ok {
		return label
	}
	return key
}

func normalizeImagePayloadOptions(payload any) {
	input, ok := payload.(map[string]any)
	if !ok || input == nil {
		return
	}

	if raw, ok := input["aspect_ratio"].(string); ok {
		if normalized := normalizePortableAspectRatio(raw); normalized != "" {
			input["aspect_ratio"] = normalized
		}
	}
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}
