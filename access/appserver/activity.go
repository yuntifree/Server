package main

import (
	"Server/httpserver"
	"Server/proto/common"
	"Server/proto/config"
	"Server/util"
	"net/http"
)

var stationRsp = `
{
	"errno":0,
	"data":{
		"infos":[
			{
				"id":1,
				"name":"东莞市中心血站",
				"timeslots":[
					"09:00~11:00",
					"11:00~13:00",
					"13:00~15:00",
					"15:00~17:00",
					"17:00~18:00"
				]
			},
			{
				"id":2,
				"name":"城区捐血中心",
				"timeslots":[
					"09:00~11:00",
					"11:00~13:00",
					"13:00~15:00",
					"15:00~17:00",
					"17:00~18:00"
				]
			},
			{
				"id":3,
				"name":"塘厦捐血站",
				"timeslots":[
					"09:00~11:00",
					"11:00~13:00",
					"13:00~15:00",
					"15:00~17:00",
					"17:00~18:00"
				]
			},
			{
				"id":4,
				"name":"人民公园爱心献血屋",
				"timeslots":[
					"10:00~11:00",
					"11:00~13:00",
					"13:00~15:00",
					"15:00~17:00",
					"17:00~18:00"
				]
			},
			{
				"id":5,
				"name":"虎门爱心献血屋",
				"timeslots":[
					"09:30~11:00",
					"11:00~13:00",
					"13:00~15:00",
					"15:00~17:00",
					"17:00~18:00"
				]
			},
			{
				"id":6,
				"name":"西平爱心献血屋",
				"timeslots":[
					"10:00~11:00",
					"11:00~13:00",
					"13:00~15:00",
					"15:00~17:00",
					"17:00~18:00"
				]
			},
			{
				"id":7,
				"name":"厚街爱心献血屋",
				"timeslots":[
					"10:00~11:00",
					"11:00~13:00",
					"13:00~15:00",
					"15:00~17:00",
					"17:00~18:00"
				]
			}

		]
	}
}

`

func getStations(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	w.Write([]byte(stationRsp))
	return nil
}

func submitReserveInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	name := req.GetParamString("name")
	phone := req.GetParamString("phone")
	date := req.GetParamString("date")
	btype := req.GetParamInt("btype")
	sid := req.GetParamInt("sid")
	pillow := req.GetParamInt("pillow")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, 0, "SubmitReserveInfo",
		&config.ReserveRequest{Head: &common.Head{Sid: uuid},
			Name: name, Phone: phone, Sid: sid, Date: date,
			Btype: btype, Pillow: pillow})
	httpserver.CheckRPCErr(rpcerr, "SubmitReserveInfo")
	res := resp.Interface().(*config.ReserveReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "SubmitReserveInfo")

	body := httpserver.GenResponseBody(res, true)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}
