package voice

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed templates/voice_start.xml
var voiceStartTemplate string

//go:embed templates/voice_find_stop.xml
var voiceFindStopTemplate string

//go:embed templates/voice_error.xml
var voiceErrorTemplate string

//go:embed templates/voice_disambiguation.xml
var voiceDisambiguationTemplate string

type TemplateManager struct {
	startTemplate          *template.Template
	findStopTemplate       *template.Template
	errorTemplate          *template.Template
	disambiguationTemplate *template.Template
}

type VoiceStartContext struct {
	WelcomePrompt string
	Language      string `json:"language,omitempty"`
}

type VoiceFindStopContext struct {
	ArrivalsMessage string
	MinutesAfter    int
	MenuPrompt      string
	Language        string `json:"language,omitempty"`
}

type VoiceErrorContext struct {
	ErrorMessage string
}

type VoiceDisambiguationContext struct {
	DisambiguationPrompt string
	Language             string `json:"language,omitempty"`
}

func NewTemplateManager() (*TemplateManager, error) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}

	startTmpl, err := template.New("voice_start").Parse(voiceStartTemplate)
	if err != nil {
		return nil, err
	}

	findStopTmpl, err := template.New("voice_find_stop").Funcs(funcMap).Parse(voiceFindStopTemplate)
	if err != nil {
		return nil, err
	}

	errorTmpl, err := template.New("voice_error").Parse(voiceErrorTemplate)
	if err != nil {
		return nil, err
	}

	disambiguationTmpl, err := template.New("voice_disambiguation").Parse(voiceDisambiguationTemplate)
	if err != nil {
		return nil, err
	}

	return &TemplateManager{
		startTemplate:          startTmpl,
		findStopTemplate:       findStopTmpl,
		errorTemplate:          errorTmpl,
		disambiguationTemplate: disambiguationTmpl,
	}, nil
}

func (tm *TemplateManager) RenderVoiceStart(ctx VoiceStartContext) (string, error) {
	var buf strings.Builder
	err := tm.startTemplate.Execute(&buf, ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (tm *TemplateManager) RenderVoiceFindStop(ctx VoiceFindStopContext) (string, error) {
	var buf strings.Builder
	err := tm.findStopTemplate.Execute(&buf, ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (tm *TemplateManager) RenderVoiceError(ctx VoiceErrorContext) (string, error) {
	var buf strings.Builder
	err := tm.errorTemplate.Execute(&buf, ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (tm *TemplateManager) RenderVoiceDisambiguation(ctx VoiceDisambiguationContext) (string, error) {
	var buf strings.Builder
	err := tm.disambiguationTemplate.Execute(&buf, ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
