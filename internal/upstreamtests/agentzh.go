package upstreamtests

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

type AgentzhCase struct {
	Name string
	Lua  string
	Out  string
}

func FindAgentzhCase(cases []AgentzhCase, name string) (AgentzhCase, bool) {
	for _, testCase := range cases {
		if testCase.Name == name {
			return testCase, true
		}
	}
	return AgentzhCase{}, false
}

func RunAgentzhCase(testCase AgentzhCase) error {
	result, err := RunLua(testCase.Lua)
	if err != nil {
		return fmt.Errorf("run lua for %q: %w", testCase.Name, err)
	}

	expected := testCase.Out
	actual := strings.TrimRight(result.Output, "\n")
	if actual != expected {
		return fmt.Errorf("output mismatch for %q: got %q want %q", testCase.Name, actual, expected)
	}

	return nil
}

func ParseAgentzhCases() ([]AgentzhCase, error) {
	data, err := ReadFixture("tests", "agentzh.t")
	if err != nil {
		return nil, err
	}

	return parseAgentzhCases(data)
}

func parseAgentzhCases(data []byte) ([]AgentzhCase, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		inData      bool
		current     *AgentzhCase
		currentPart string
		cases       []AgentzhCase
	)

	flush := func() {
		if current == nil {
			return
		}
		current.Lua = strings.TrimRight(current.Lua, "\n")
		current.Out = strings.TrimRight(current.Out, "\n")
		cases = append(cases, *current)
		current = nil
		currentPart = ""
	}

	for scanner.Scan() {
		line := scanner.Text()

		if !inData {
			if line == "__DATA__" {
				inData = true
			}
			continue
		}

		if strings.HasPrefix(line, "===") {
			flush()
			current = &AgentzhCase{Name: strings.TrimSpace(strings.TrimPrefix(line, "==="))}
			continue
		}

		if current == nil {
			continue
		}

		if strings.HasPrefix(line, "--- ") {
			currentPart = strings.TrimSpace(strings.TrimPrefix(line, "--- "))
			continue
		}

		switch currentPart {
		case "lua":
			current.Lua += line + "\n"
		case "out":
			current.Out += line + "\n"
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan agentzh cases: %w", err)
	}

	flush()
	if len(cases) == 0 {
		return nil, fmt.Errorf("no agentzh cases parsed")
	}

	return cases, nil
}
