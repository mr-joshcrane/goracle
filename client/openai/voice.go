package openai

//
import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

type Voice string

const (
	TTS     = "tts-1"
	TTS_HQ  = "tts-1-hq"
	WHISPER = "whisper-1"
)

const (
	Alloy   Voice = "alloy"
	Echo    Voice = "echo"
	Fable   Voice = "fable"
	Onyx    Voice = "onyx"
	Nova    Voice = "nova"
	Shimmer Voice = "shimmer"
)

type TextToSpeechRequestBody struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice Voice  `json:"voice"`
}

type TTSReqOptions func(TextToSpeechRequestBody) TextToSpeechRequestBody

func WithVoice(voice Voice) TTSReqOptions {
	return func(req TextToSpeechRequestBody) TextToSpeechRequestBody {
		req.Voice = voice
		return req
	}
}

func TextToSpeech(ctx context.Context, token string, text string) ([]byte, error) {
	req, err := CreateTextToSpeechRequest(token, text)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, NewClientError(resp)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func SpeechToText(ctx context.Context, token string, audio []byte) (string, error) {
	req, err := CreateSpeechToTextRequest(token, audio)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", NewClientError(resp)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func CreateTextToSpeechRequest(token string, text string, opts ...TTSReqOptions) (*http.Request, error) {
	request := TextToSpeechRequestBody{
		Model: TTS,
		Input: text,
		Voice: Echo,
	}
	for _, opt := range opts {
		request = opt(request)
	}
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/audio/speech", buf)
	if err != nil {
		return nil, err
	}
	req = addDefaultHeaders(token, req)
	return req, nil
}

func CreateSpeechToTextRequest(token string, audio []byte) (*http.Request, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	err := writer.WriteField("model", WHISPER)
	if err != nil {
		return nil, err
	}
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return nil, err
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", "audio/wav")
	if _, err := part.Write(audio); err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/audio/transcriptions", buf)
	if err != nil {
		return nil, err
	}
	req = addDefaultHeaders(token, req)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
	return req, nil
}
