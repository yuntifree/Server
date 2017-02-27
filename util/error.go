package util

//AppError for app error handler
type AppError struct {
	Code     int
	Msg      string
	Callback string
}

//Error return error message
func (err *AppError) Error() string {
	return err.Msg
}
