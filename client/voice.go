package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func (c *ChatGPT) textToSpeech(ctx context.Context, transform Transform) error {
	text, err := io.ReadAll(transform.GetSource())
	if err != nil {
		return err
	}
	req, err := CreateTextToSpeechRequest(c.Token, string(text))
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("bad status code: %d, %s", resp.StatusCode, string(data))
	}
	target := transform.GetTarget()
	_, err = io.Copy(target, resp.Body)
	return err
}

func (c *ChatGPT) speechToText(ctx context.Context, transform Transform) error {
	audio, err := io.ReadAll(transform.GetSource())
	if err != nil {
		return err
	}

	req, err := CreateSpeechToTextRequest(c.Token, audio)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("bad status code: %d, %s", resp.StatusCode, string(data))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_, err = transform.GetTarget().Write(data)
	return err
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

func chunkify(data string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}
