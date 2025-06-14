package formatters

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

type VoiceTemplateManager struct {
	startTemplate          *template.Template
	findStopTemplate       *template.Template
	errorTemplate          *template.Template
	disambiguationTemplate *template.Template
}

type VoiceStartContext struct {
	WelcomePrompt string
}

type VoiceFindStopContext struct {
	ArrivalsMessage string
}

type VoiceErrorContext struct {
	ErrorMessage string
}

type VoiceDisambiguationContext struct {
	DisambiguationPrompt string
}

func NewVoiceTemplateManager() (*VoiceTemplateManager, error) {
	startTmpl, err := template.New("voice_start").Parse(voiceStartTemplate)
	if err != nil {
		return nil, err
	}

	findStopTmpl, err := template.New("voice_find_stop").Parse(voiceFindStopTemplate)
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

	return &VoiceTemplateManager{
		startTemplate:          startTmpl,
		findStopTemplate:       findStopTmpl,
		errorTemplate:          errorTmpl,
		disambiguationTemplate: disambiguationTmpl,
	}, nil
}

func (vtm *VoiceTemplateManager) RenderVoiceStart(ctx VoiceStartContext) (string, error) {
	var buf strings.Builder
	err := vtm.startTemplate.Execute(&buf, ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (vtm *VoiceTemplateManager) RenderVoiceFindStop(ctx VoiceFindStopContext) (string, error) {
	var buf strings.Builder
	err := vtm.findStopTemplate.Execute(&buf, ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (vtm *VoiceTemplateManager) RenderVoiceError(ctx VoiceErrorContext) (string, error) {
	var buf strings.Builder
	err := vtm.errorTemplate.Execute(&buf, ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (vtm *VoiceTemplateManager) RenderVoiceDisambiguation(ctx VoiceDisambiguationContext) (string, error) {
	var buf strings.Builder
	err := vtm.disambiguationTemplate.Execute(&buf, ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}