package logger

type Logger interface {
	Info(message string)
	Warn(message string)
	Error(message string, err error)
}
