package controllers

import "strings"

// VoiceInfo describes a Qwen TTS voice
type VoiceInfo struct {
	Value       string   `json:"value"`       // API voice parameter, e.g. "Cherry"
	Label       string   `json:"label"`       // display name
	Description string   `json:"description"` // short description
	Languages   []string `json:"languages"`   // supported languages
}

// ModelVoiceMap maps model family -> supported voice list
// Note: voices are grouped by model "family", e.g. qwen3-tts-flash* is one group, qwen-tts* is another.
var ModelVoiceMap = map[string][]VoiceInfo{
	// Qwen3-TTS-Flash series (qwen3-tts-flash / qwen3-tts-flash-2025-11-27 / qwen3-tts-flash-2025-09-18)
	"qwen3-tts-flash": {
		{Value: "Cherry", Label: "Cherry", Description: "Sunny and positive, warm and natural female voice (female)"},
		{Value: "Serena", Label: "Serena", Description: "Gentle female voice (female)"},
		{Value: "Ethan", Label: "Ethan", Description: "Standard Mandarin with slight northern accent, sunny and energetic (male)"},
		{Value: "Chelsie", Label: "Chelsie", Description: "2D virtual girlfriend (female)"},
		{Value: "Momo", Label: "Momo", Description: "Playful and funny, here to cheer you up (female)"},
		{Value: "Vivian", Label: "Vivian", Description: "Cool and cute with a little attitude (female)"},
		{Value: "Moon", Label: "Moon", Description: "Confident and stylish (male)"},
		{Value: "Maia", Label: "Maia", Description: "Intellectual meets gentle (female)"},
		{Value: "Kai", Label: "Kai", Description: "A spa for your ears (male)"},
		{Value: "Nofish", Label: "Nofish", Description: "A designer who cannot pronounce retroflex consonants (male)"},
		{Value: "Bella", Label: "Bella", Description: "Soft and sweet little girl voice (female)"},
		{Value: "Jennifer", Label: "Jennifer", Description: "Brand-quality, cinematic American female voice (female)"},
		{Value: "Ryan", Label: "Ryan", Description: "Full of rhythm, explosive performance, dance of reality and tension (male)"},
		{Value: "Katerina", Label: "Katerina", Description: "Elegant female voice with lingering rhythm (female)"},
		{Value: "Aiden", Label: "Aiden", Description: "American big boy who is skilled in cooking (male)"},
		{Value: "Eldric Sage", Label: "Eldric Sage", Description: "Calm and wise elder, weathered like pine yet clear-minded (male)"},
		{Value: "Mia", Label: "Mia", Description: "Gentle as spring water, obedient as first snow (female)"},
		{Value: "Mochi", Label: "Mochi", Description: "Smart little adult, childlike yet precocious as Zen (male)"},
		{Value: "Bellona", Label: "Bellona", Description: "Loud voice, clear articulation, vivid character (female)"},
		{Value: "Vincent", Label: "Vincent", Description: "Hoarse smoky voice, telling stories of thousands of troops and rivers (male)"},
		{Value: "Bunny", Label: "Bunny", Description: "Super cute little girl overflowing with cuteness (female)"},
		{Value: "Neil", Label: "Neil", Description: "The most professional news anchor (male)"},
		{Value: "Elias", Label: "Elias", Description: "Rigorous yet narrative instructor voice (female)"},
		{Value: "Arthur", Label: "Arthur", Description: "Simple and honest voice soaked in years and tobacco (male)"},
		{Value: "Nini", Label: "Nini", Description: "Soft and sticky voice like glutinous rice cake (female)"},
		{Value: "Ebona", Label: "Ebona", Description: "Slightly horror-style grandmother voice (female)"},
		{Value: "Seren", Label: "Seren", Description: "Warm and soothing, sleep-aid voice (female)"},
		{Value: "Pip", Label: "Pip", Description: "Naughty but full of childlike innocence (male)"},
		{Value: "Stella", Label: "Stella", Description: "Usually sweet as candy, full of justice at critical moments (female)"},
		{Value: "Bodega", Label: "Bodega", Description: "Enthusiastic Spanish uncle (male)"},
		{Value: "Sonrisa", Label: "Sonrisa", Description: "Warm and outgoing Latin American big sister (female)"},
		{Value: "Alek", Label: "Alek", Description: "Warm voice beneath the cold exterior of a Slavic man (male)"},
		{Value: "Dolce", Label: "Dolce", Description: "Lazy Italian uncle (male)"},
		{Value: "Sohee", Label: "Sohee", Description: "Gentle, cheerful, and emotionally rich Korean voice (female)"},
		{Value: "Ono Anna", Label: "Ono Anna", Description: "Quirky childhood sweetheart (female)"},
		{Value: "Lenn", Label: "Lenn", Description: "German youth with rationality as base and rebellion hidden in details (male)"},
		{Value: "Emilien", Label: "Emilien", Description: "Romantic French big brother (male)"},
		{Value: "Andre", Label: "Andre", Description: "Magnetic, natural, steady male voice (male)"},
		{Value: "Radio Gol", Label: "Radio Gol", Description: "Football poet-style commentary (male)"},
		{Value: "Jada", Label: "Shanghai-Zhen", Description: "Energetic Shanghai big sister (female)"},
		{Value: "Dylan", Label: "Beijing-Dong", Description: "Young man grown up in Beijing hutong (male)"},
		{Value: "Li", Label: "Nanjing-Lao Li", Description: "Patient yoga teacher (male)"},
		{Value: "Marcus", Label: "Shaanxi-Qin Chuan", Description: "Full of old Shaanxi flavor (male)"},
		{Value: "Roy", Label: "Minnan-A Jie", Description: "Humorous and straightforward Taiwanese boy (male)"},
		{Value: "Peter", Label: "Tianjin-Li Peter", Description: "Tianjin cross-talk professional straight man (male)"},
		{Value: "Sunny", Label: "Sichuan-Qing'er", Description: "Sweet Sichuan girl (female)"},
		{Value: "Eric", Label: "Sichuan-Cheng Chuan", Description: "Lively Chengdu man from Sichuan (male)"},
		{Value: "Rocky", Label: "Cantonese-A Qiang", Description: "Humorous and witty A Qiang (male)"},
		{Value: "Kiki", Label: "Cantonese-A Qing", Description: "Sweet Hong Kong girl bestie (female)"},
	},

	// Qwen-TTS series (qwen-tts / qwen-tts-latest / qwen-tts-2025-xx-xx)
	"qwen-tts": {
		{Value: "Cherry", Label: "Cherry", Description: "Sunny and positive, warm and natural female voice (female)"},
		{Value: "Serena", Label: "Serena", Description: "Gentle female voice (female)"},
		{Value: "Ethan", Label: "Ethan", Description: "Standard Mandarin with slight northern accent, sunny and energetic (male)"},
		{Value: "Chelsie", Label: "Chelsie", Description: "2D virtual girlfriend (female)"},
		{Value: "Momo", Label: "Momo", Description: "Playful and funny, here to cheer you up (female)"},
		// additional voices can be added as needed
	},
}

// normalizeModel normalizes a specific model name to a model family key
// e.g.: qwen3-tts-flash-2025-11-27 -> qwen3-tts-flash
//
//	qwen-tts-2025-05-22       -> qwen-tts
func normalizeModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if strings.HasPrefix(model, "qwen3-tts-flash") {
		return "qwen3-tts-flash"
	}
	if strings.HasPrefix(model, "qwen-tts") {
		return "qwen-tts"
	}
	return model
}

// GetVoicesByModel returns supported voices for the given model name
func GetVoicesByModel(model string) []VoiceInfo {
	key := normalizeModel(model)
	if voices, ok := ModelVoiceMap[key]; ok {
		return voices
	}
	return nil
}

// IsVoiceSupported checks whether a given voice is supported by the specified model
func IsVoiceSupported(model, voice string) bool {
	if voice == "" {
		return false
	}
	for _, v := range GetVoicesByModel(model) {
		if v.Value == voice {
			return true
		}
	}
	return false
}
