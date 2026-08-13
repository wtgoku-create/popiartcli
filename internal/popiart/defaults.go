package popiart

const DefaultVoiceID = "male-qn-qingse"

var defaultModelCandidates = map[string][]string{
	"image":                 {"Nano-banana-pro", "gemini-3-pro-image-preview", "seedream-4-5-251128"},
	"image.generate":        {"Nano-banana-pro", "gemini-3-pro-image-preview", "seedream-4-5-251128"},
	"image.img2img":         {"Nano-banana-pro", "gemini-3-pro-image-preview", "seedream-4-5-251128"},
	"image.transform":       {"Nano-banana-pro", "gemini-3-pro-image-preview", "seedream-4-5-251128"},
	"image.describe":        {"Doubao seed 2.0 lite"},
	"video":                 {"viduq2-pro", "viduq2-pro-fast"},
	"video.generate":        {"viduq2-pro", "viduq2-pro-fast"},
	"video.img2video":       {"viduq2-pro", "viduq2-pro-fast"},
	"video.from-image":      {"viduq2-pro", "viduq2-pro-fast"},
	"video.seedance":        {"huimeng-seedance-2.0"},
	"video.action-transfer": {"jimeng_dreamactor_m20_gen_video"},
	"audio.tts":             {"speech-2.8-hd"},
	"speech.synthesize":     {"speech-2.8-hd"},
	"music.generate":        {"music-2.6", "music-2.6-free"},
	"music":                 {"music-2.6", "music-2.6-free"},
}

// DefaultModelCodes 返回命令对应的默认 aiModelCode 候选列表。
func DefaultModelCodes(command string) []string {
	values := defaultModelCandidates[command]
	out := make([]string, 0, len(values))
	out = append(out, values...)
	return out
}
