package notify

func send(title, body string) error {
	return run("notify-send", title, body)
}
