package oracle_test

import (
	"image"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mr-joshcrane/oracle"
)

func TestGeneratePrompt_GeneratesExpectedPrompt(t *testing.T) {
	t.Parallel()
	o := oracle.NewOracle("dummy-token-openai")
	o.SetPurpose("To answer if a number is odd or even in a specific format")
	o.GiveExample("2", "+++even+++")
	o.GiveExample("3", "---odd---")
	got := o.GeneratePrompt("4")
	want := oracle.Prompt{
		Purpose: "To answer if a number is odd or even in a specific format",
		ExampleInputs: []string{
			"2",
			"3",
		},
		IdealOutputs: []string{
			"+++even+++",
			"---odd---",
		},
		Question: "4",
		Images:   []image.Image{},
		Urls:     []url.URL{},
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestWithVision_ImageModality(t *testing.T) {
	t.Parallel()
	o := oracle.NewOracle("dummy-token-openai")
	o.SetPurpose("To detect if there is a human in the image")
	testImage := image.NewGray(image.Rect(0, 0, 1, 1))
	got := o.GeneratePrompt("Is there a human in this image?", oracle.NewImage(testImage))
	want := oracle.Prompt{
		Purpose:  "To detect if there is a human in the image",
		Question: "Is there a human in this image?",
		Images:   []image.Image{testImage},
		Urls:     []url.URL{},
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestWithVision_UrlModality(t *testing.T) {
	t.Parallel()
	o := oracle.NewOracle("dummy-token-openai")
	o.SetPurpose("To detect if there is a human in the image")
	testUrl, _ := url.Parse("https://www.google.com")
	got := o.GeneratePrompt("Is there a human in this image?", oracle.NewURL(*testUrl))
	want := oracle.Prompt{
		Purpose:  "To detect if there is a human in the image",
		Question: "Is there a human in this image?",
		Images:   []image.Image{},
		Urls:     []url.URL{*testUrl},
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestWithVision_MixedModality(t *testing.T) {
	t.Parallel()
	o := oracle.NewOracle("dummy-token-openai")
	o.SetPurpose("To detect if there is a human in the image")
	testImage := image.NewGray(image.Rect(0, 0, 1, 1))
	testUrl, _ := url.Parse("https://www.google.com")
	got := o.GeneratePrompt("Is there a human in this image?", oracle.NewImage(testImage), oracle.NewURL(*testUrl))
	want := oracle.Prompt{
		Purpose:  "To detect if there is a human in the image",
		Question: "Is there a human in this image?",
		Images:   []image.Image{testImage},
		Urls:     []url.URL{*testUrl},
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	o := oracle.NewOracle("dummy-token-openai")
	o.SetPurpose("Can you remember what I had to say?")
	o.GiveExample("I can remember", "I can remember")
	got := o.GeneratePrompt("What did I have to say?")
	want := oracle.Prompt{
		Purpose:       "Can you remember what I had to say?",
		ExampleInputs: []string{"I can remember"},
		IdealOutputs:  []string{"I can remember"},
		Question:      "What did I have to say?",
		Images:        []image.Image{},
		Urls:          []url.URL{},
	}
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
	o.Reset()
	got = o.GeneratePrompt("What about after the reset?")
	want = oracle.Prompt{
		Purpose:       "",
		ExampleInputs: []string{},
		IdealOutputs:  []string{},
		Question:      "What about after the reset?",
		Images:        []image.Image{},
		Urls:          []url.URL{},
	}
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestAccessorMethodsOnPrompt(t *testing.T) {
	t.Parallel()
	p := oracle.Prompt{
		Purpose:       "Respond with lame witicisms",
		ExampleInputs: []string{"How do you eat an elephant?", "What is the essence of humour?"},
		IdealOutputs:  []string{"One bite at a time", "Timing"},
		Question:      "Where do you find an elephant?",
		Images: []image.Image{
			image.NewGray(image.Rect(0, 0, 1, 1)),
		},
		Urls: []url.URL{
			*(&url.URL{
				Scheme: "https",
				Host:   "www.google.com",
			}),
		},
	}
	cases := []struct {
		description string
		got         any
		want        any
	}{
		{
			description: "Purpose",
			got:         p.GetPurpose(),
			want:        "Respond with lame witicisms",
		},
		{
			description: "Examples",
			got:         func() []string { example, _ := p.GetExamples(); return example }(),
			want:        []string{"How do you eat an elephant?", "What is the essence of humour?"},
		},
		{
			description: "Ideals",
			got:         func() []string { _, ideal := p.GetExamples(); return ideal }(),
			want:        []string{"One bite at a time", "Timing"},
		},
		{
			description: "Question",
			got:         p.GetQuestion(),
			want:        "Where do you find an elephant?",
		},
		{
			description: "Images",
			got:         p.GetImages(),
			want: []image.Image{
				image.NewGray(image.Rect(0, 0, 1, 1)),
			},
		},
		{
			description: "Urls",
			got:         p.GetUrls(),
			want:        func() []url.URL { u, _ := url.Parse("https://www.google.com"); return []url.URL{*u} }(),
		},
	}
	for _, c := range cases {
		if !cmp.Equal(c.want, c.got) {
			t.Error(cmp.Diff(c.want, c.got))
		}
	}
}
