package deskact

// DefaultSpecialKeys returns the default special-key mapping.
func DefaultSpecialKeys() map[string]string {
	return map[string]string{
		"~": "`",
		"!": "1",
		"@": "2",
		"#": "3",
		"$": "4",
		"%": "5",
		"^": "6",
		"&": "7",
		"*": "8",
		"(": "9",
		")": "0",
		"_": "-",
		"+": "=",
		"{": "[",
		"}": "]",
		"|": "\\",
		":": ";",
		`"`: "'",
		"<": ",",
		">": ".",
		"?": "/",
	}
}

func lookupSpecialKey(key string) (string, bool) {
	switch key {
	case "~":
		return "`", true
	case "!":
		return "1", true
	case "@":
		return "2", true
	case "#":
		return "3", true
	case "$":
		return "4", true
	case "%":
		return "5", true
	case "^":
		return "6", true
	case "&":
		return "7", true
	case "*":
		return "8", true
	case "(":
		return "9", true
	case ")":
		return "0", true
	case "_":
		return "-", true
	case "+":
		return "=", true
	case "{":
		return "[", true
	case "}":
		return "]", true
	case "|":
		return "\\", true
	case ":":
		return ";", true
	case `"`:
		return "'", true
	case "<":
		return ",", true
	case ">":
		return ".", true
	case "?":
		return "/", true
	default:
		return "", false
	}
}
