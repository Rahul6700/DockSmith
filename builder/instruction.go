package builder

// somewhat like enum
// makes sure it accepts only predefined strings
type InstructionType string

const (
	COPY InstructionType = "COPY"
)

// if my inst is COPY . /app then,
// type -> COPY
// Args -> [".", "/app"]
type Instruction struct {
	Type InstructionType // enum string
	Args []string
}
