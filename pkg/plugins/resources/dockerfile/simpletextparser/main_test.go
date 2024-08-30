package simpletextparser

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/updatecli/updatecli/pkg/core/result"
	"github.com/updatecli/updatecli/pkg/plugins/resources/dockerfile/simpletextparser/keywords"
	"github.com/updatecli/updatecli/pkg/plugins/resources/dockerfile/types"
)

func TestInstruction_setKeywordLogic(t *testing.T) {
	tests := []struct {
		name             string
		parser           SimpleTextDockerfileParser
		wantLogic        keywords.Logic
		wantErrorMessage string
	}{
		{
			name: "'FROM' instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "FROM",
			},
			wantLogic: keywords.From{},
		},
		{
			name: "'from' instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "from",
			},
			wantLogic: keywords.From{},
		},
		{
			name: "'ARG' instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "ARG",
			},
			wantLogic: keywords.Arg{},
		},
		{
			name: "'arg' instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "arg",
			},
			wantLogic: keywords.Arg{},
		},
		{
			name: "'ENV' instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "ENV",
			},
			wantLogic: keywords.Env{},
		},
		{
			name: "'env' instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "env",
			},
			wantLogic: keywords.Env{},
		},
		{
			name: "'LABEL' instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "LABEL",
			},
			wantLogic: keywords.Label{},
		},
		{
			name: "'label' instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "label",
			},
			wantLogic: keywords.Label{},
		},
		{
			name: "Not supported (yet) instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "ONBUILD",
			},
			wantLogic:        nil,
			wantErrorMessage: fmt.Sprintf("%s Provided keyword \"ONBUILD\" not supported (yet). Feel free to open an issue explaining your use-case to help adding the implementation.", result.FAILURE),
		},
		{
			name: "Unknown instruction",
			parser: SimpleTextDockerfileParser{
				Keyword: "QUACK",
			},
			wantLogic:        nil,
			wantErrorMessage: fmt.Sprintf("%s Unknown keyword \"QUACK\" provided for Dockerfile's instruction.", result.FAILURE),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parserUndertest := tt.parser
			err := parserUndertest.setKeywordLogic()

			if tt.wantErrorMessage != "" {
				assert.Equal(t, tt.wantErrorMessage, err.Error())
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tt.wantLogic, parserUndertest.KeywordLogic)
		})
	}
}

func TestSimpleTextDockerfileParser_FindInstruction(t *testing.T) {
	tests := []struct {
		name              string
		fixtureDockerfile string
		givenInstruction  map[string]string
		expected          bool
	}{
		{
			name:              "Find Instruction FROM golang",
			fixtureDockerfile: "FROM.Dockerfile",
			expected:          true,
			givenInstruction: map[string]string{
				"keyword": "FROM",
				"matcher": "golang",
			},
		},
		{
			name:              "Find Instruction from golang",
			fixtureDockerfile: "FROM.Dockerfile",
			expected:          true,
			givenInstruction: map[string]string{
				"keyword": "from",
				"matcher": "golang",
			},
		},
		{
			name:              "Find Instruction from GOLANG",
			fixtureDockerfile: "FROM.Dockerfile",
			expected:          false,
			givenInstruction: map[string]string{
				"keyword": "from",
				"matcher": "GOLANG",
			},
		},
		{
			name:              "Find Instruction ARG HELM_VERSION",
			fixtureDockerfile: "FROM.Dockerfile",
			expected:          false,
			givenInstruction: map[string]string{
				"keyword": "ARG",
				"matcher": "HELM_VERSION",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parserUnderTest, err := NewSimpleTextDockerfileParser(tt.givenInstruction)
			if err != nil {
				t.Errorf("Error while calling NewSimpleTextDockerfileParser with argument %v.", tt.givenInstruction)
			}

			originalDockerfile, err := os.ReadFile("./test_fixtures/" + tt.fixtureDockerfile)
			if err != nil {
				t.Errorf("Error while reading file %q: %v", tt.fixtureDockerfile, err)
			}

			got := parserUnderTest.FindInstruction(originalDockerfile)

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSimpleTextDockerfileParser_ReplaceInstruction(t *testing.T) {
	tests := []struct {
		name                 string
		fixtureDockerfile    string
		givenSource          string
		givenInstruction     map[string]string
		expectedChanges      types.ChangedLines
		expectedErrorMessage string // If undefined (e.g. equals to empty string), then no error is expected
	}{
		{
			name:              "Change golang FROM instruction",
			fixtureDockerfile: "FROM.Dockerfile",
			givenSource:       "1.16.0-alpine",
			givenInstruction: map[string]string{
				"keyword": "FROM",
				"matcher": "golang",
			},
			expectedChanges: types.ChangedLines{
				1: types.LineDiff{
					Original: "FROM golang:1.15 AS builder",
					New:      "FROM golang:1.16.0-alpine AS builder",
				},
				11: types.LineDiff{
					Original: "FROM golang:1.15 as tester",
					New:      "FROM golang:1.16.0-alpine as tester",
				},
				15: types.LineDiff{
					Original: "FROM golang AS reporter",
					New:      "FROM golang:1.16.0-alpine AS reporter",
				},
				19: types.LineDiff{
					Original: "FROM golang",
					New:      "FROM golang:1.16.0-alpine",
				},
			},
		},
		{
			name:              "Change HELM_VERSION ARG instruction",
			fixtureDockerfile: "ARG.Dockerfile",
			givenSource:       "4.5.6",
			givenInstruction: map[string]string{
				"keyword": "ARG",
				"matcher": "HELM_VERSION",
			},
			expectedChanges: types.ChangedLines{
				3: types.LineDiff{
					Original: "ARG HELM_VERSION=3.0.0",
					New:      "ARG HELM_VERSION=4.5.6",
				},
				17: types.LineDiff{
					Original: "arg HELM_VERSION=3.0.0",
					New:      "arg HELM_VERSION=4.5.6",
				},
				27: types.LineDiff{
					Original: "ARG HELM_VERSION",
					New:      "ARG HELM_VERSION=4.5.6",
				},
			},
		},
		{
			name:              "Instruction not matched",
			fixtureDockerfile: "FROM.Dockerfile",
			givenSource:       "4.5.6",
			givenInstruction: map[string]string{
				"keyword": "ARG",
				"matcher": "HELM_VERSION",
			},
			expectedChanges:      types.ChangedLines{},
			expectedErrorMessage: fmt.Sprintf("%s No line found matching the keyword \"ARG\" and the matcher \"HELM_VERSION\".", result.FAILURE),
		},
		{
			name:              "Instruction kept the same",
			fixtureDockerfile: "FROM.Dockerfile",
			givenSource:       "3.0.0",
			givenInstruction: map[string]string{
				"keyword": "ARG",
				"matcher": "golang",
			},
			expectedChanges: types.ChangedLines{},
		},
		{
			name:              "Change org.opencontainers.image.version LABEL instruction",
			fixtureDockerfile: "LABEL.Dockerfile",
			givenSource:       "0.14.1",
			givenInstruction: map[string]string{
				"keyword": "LABEL",
				"matcher": "org.opencontainers.image.version",
			},
			expectedChanges: types.ChangedLines{
				3: types.LineDiff{
					Original: "LABEL org.opencontainers.image.version=0.14.0",
					New:      "LABEL org.opencontainers.image.version=0.14.1",
				},
				7: types.LineDiff{
					Original: "label org.opencontainers.image.version=0.14.0",
					New:      "label org.opencontainers.image.version=0.14.1",
				},
				11: types.LineDiff{
					Original: "LABEL org.opencontainers.image.version",
					New:      "LABEL org.opencontainers.image.version=0.14.1",
				},
				19: types.LineDiff{
					Original: "LABEL org.opencontainers.image.version",
					New:      "LABEL org.opencontainers.image.version=0.14.1",
				},
				20: types.LineDiff{
					Original: "label org.opencontainers.image.version=0.14.0",
					New:      "label org.opencontainers.image.version=0.14.1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parserUnderTest, err := NewSimpleTextDockerfileParser(tt.givenInstruction)
			if err != nil {
				t.Errorf("Error while calling NewSimpleTextDockerfileParser with argument %v.", tt.givenInstruction)
			}

			originalDockerfile, err := os.ReadFile("./test_fixtures/" + tt.fixtureDockerfile)
			if err != nil {
				t.Errorf("Error while reading file %q: %v", tt.fixtureDockerfile, err)
			}

			gotContent, gotChanges, gotError := parserUnderTest.ReplaceInstructions(originalDockerfile, tt.givenSource)

			// Check for expected errors
			if tt.expectedErrorMessage != "" {
				assert.NotNil(t, gotError)
				assert.Equal(t, tt.expectedErrorMessage, gotError.Error())
			}
			// Check for expected changes
			assert.Equal(t, tt.expectedChanges, gotChanges)

			// Check for the resulting dockerfilecontent
			originalLines := strings.Split(string(originalDockerfile), "\n")
			gotLines := strings.Split(string(gotContent), "\n")

			// Both content should have the same number of line as only replacements are expected
			assert.Equal(t, len(originalLines), len(gotLines))

			// The diff between fixture and resulting tmpfile must match the tt.wantedChanges
			for index, currentLine := range originalLines {
				lineDiff, exists := tt.expectedChanges[index+1]
				if exists {
					// current's line number found in wantDiff: currentLine should have been changed
					assert.Equal(t, lineDiff.Original, currentLine)
					assert.Equal(t, lineDiff.New, gotLines[index])
				} else {
					// current's line number NOT found in wantDiff lines should be the same
					assert.Equal(t, currentLine, gotLines[index])
				}
			}
		})
	}
}

func TestSimpleTextDockerfileParser_GetInstruction(t *testing.T) {
	tests := []struct {
		name              string
		fixtureDockerfile string
		givenInstruction  map[string]string
		expectedResult    []types.StageInstructionValue
	}{
		{
			name:              "parse ARG Dockerfile",
			fixtureDockerfile: "ARG.Dockerfile",
			givenInstruction: map[string]string{
				"keyword": "ARG",
				"matcher": "HELM_VERSION",
			},
			expectedResult: []types.StageInstructionValue{
				types.StageInstructionValue{StageName: "builder", Value: "3.0.0"},
				types.StageInstructionValue{StageName: "tester", Value: "3.0.0"},
				types.StageInstructionValue{StageName: "reporter", Value: ""},
			},
		},
		{
			name:              "parse Arg Dockerfile (lowercase)",
			fixtureDockerfile: "ARG.Dockerfile",
			givenInstruction: map[string]string{
				"keyword": "arg",
				"matcher": "helm_version",
			},
			expectedResult: []types.StageInstructionValue{
				types.StageInstructionValue{StageName: "base", Value: ""},
				types.StageInstructionValue{StageName: "6", Value: ""},
			},
		},
		{
			name:              "parse Env Dockerfile",
			fixtureDockerfile: "ENV.Dockerfile",
			givenInstruction: map[string]string{
				"keyword": "ENV",
				"matcher": "TERRAFORM_VERSION",
			},
			expectedResult: []types.StageInstructionValue{
				types.StageInstructionValue{StageName: "normal-upper-case-equal", Value: "0.14.0"},
				types.StageInstructionValue{StageName: "normal-upper-case-space", Value: "\"0.14.0\""},
				types.StageInstructionValue{StageName: "normal-lower-case-equal", Value: "0.14.0"},
				types.StageInstructionValue{StageName: "default-multi-instructions-first", Value: "0.14.0"},
			},
		},
		{
			name:              "Overwriting label",
			fixtureDockerfile: "LABEL.Dockerfile",
			givenInstruction: map[string]string{
				"keyword": "LABEL",
				"matcher": "org.opencontainers.image.version",
			},
			expectedResult: []types.StageInstructionValue{
				types.StageInstructionValue{StageName: "builder", Value: "0.14.0"},
				types.StageInstructionValue{StageName: "tester", Value: "0.14.0"},
				types.StageInstructionValue{StageName: "reporter", Value: ""},
				types.StageInstructionValue{StageName: "base", Value: "0.14.0"},
			},
		},
		{
			name:              "Instruction not matched",
			fixtureDockerfile: "ARG.Dockerfile",
			givenInstruction: map[string]string{
				"keyword": "ARG",
				"matcher": "CUSTOM_VERSION",
			},
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parserUnderTest, err := NewSimpleTextDockerfileParser(tt.givenInstruction)
			if err != nil {
				t.Errorf("Error while calling NewSimpleTextDockerfileParser with argument %v.", tt.givenInstruction)
			}

			originalDockerfile, err := os.ReadFile("./test_fixtures/" + tt.fixtureDockerfile)
			if err != nil {
				t.Errorf("Error while reading file %q: %v", tt.fixtureDockerfile, err)
			}

			got := parserUnderTest.GetInstruction(originalDockerfile)

			assert.Equal(t, tt.expectedResult, got)
		})
	}
}
