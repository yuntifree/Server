package util

//ErrorType for eror
type ErrorType int32

const (
	//ParamErr get input param failed
	ParamErr ErrorType = 1
	//DataErr database query failed
	DataErr ErrorType = 2
	//JSONErr handle json data failed
	JSONErr ErrorType = 3
	//RPCErr rpc handler error
	RPCErr ErrorType = 4
	//LogicErr logic error
	LogicErr ErrorType = 5
)

//AppError for app error handler
type AppError struct {
	Type ErrorType
	Code int
	Msg  string
}

//Error return error message
func (err *AppError) Error() string {
	return err.Msg
}

//ParamError for input param error
type ParamError struct {
	Val string
}

//Error return error message
func (err *ParamError) Error() string {
	return "get param: " + err.Val + " failed"
}
