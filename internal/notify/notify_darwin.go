package notify

import "fmt"

func send(title, body string) error {
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	return run("osascript", "-e", script)
}
