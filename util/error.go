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
	Val  string
	Msg  string
}

//Error return error message
func (err *AppError) Error() string {
	if err.Type == ParamErr {
		return "get param:" + err.Val + " failed"
	}
	return err.Msg
}
