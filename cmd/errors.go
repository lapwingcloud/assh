package cmd

type invalidCommandError struct {
	message string
	Help    string
}

func (err *invalidCommandError) Error() string {
	return err.message
}

func newInvalidCommandError() error {
	return &invalidCommandError{
		"invalid command",
		`Usage:
	assh instanceId
	assh environment role [profile]

For example:
	assh i-036e822ed4ec8c585
	assh dev gaf_appserver
	assh dev gaf_appserver php72`,
	}
}
