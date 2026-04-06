package builder

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ParseDocksmithfile(path string) ([]Instruction, error) {
	
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// arr of struct Instruction -> has an enum string (inst) and an arr of args
	var instructions []Instruction

	// scanner we use to scan the file line by line
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		// strings.TrimSpace removes whitespaces from end and begin of each line
		line := strings.TrimSpace(scanner.Text())

		// if line is empty or starts with a "#", skip the line
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// strings.Fields(line) splits the string into an arr based on whitespaces
		parts := strings.Fields(line) // ex -> "COPY src.txt dest.txt" becomes ["COPY", "src.txt", "dest.txt"]
		if len(parts) == 0 {
			continue
		}
	
		// switch case for the first element of the arr (the inst basically), eg -> COPY, RUN, INSTALL
		switch parts[0] {

		case "COPY":
			if len(parts) != 3 {
				return nil, fmt.Errorf("invalid COPY at line %d", lineNum)
			}
			instructions = append(instructions, Instruction{
				Type: COPY,
				Args: parts[1:],
			})

		default:
			return nil, fmt.Errorf("unknown instruction '%s' at line %d", parts[0], lineNum)
		}
	}

	return instructions, nil
}
