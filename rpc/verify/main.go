package main

import (
	"errors"
	"log"
	"net"

	"database/sql"

	verify "../../proto/verify"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port = ":50052"
)

type server struct{}

func checkPhoneCode(phone string, code int32) (bool, error) {
	db, err := sql.Open("mysql", "root:@/yunti?charset=utf8")
	if err != nil {
		return false, err
	}

	var realcode int32
	var pid int32
	err = db.QueryRow("SELECT code, pid FROM phone_code WHERE phone = ? AND used = 0 ORDER BY pid DESC LIMIT 1", phone).Scan(&realcode, &pid)
	if err != nil {
		return false, err
	}

	if realcode == code {
		stmt, err := db.Prepare("UPDATE phone_code SET used = 1 WHERE pid = ?")
		if err != nil {
			log.Printf("update phone_code failed:%v", err)
			return true, nil
		}
		_, err = stmt.Exec(pid)
		if err != nil {
			log.Printf("update phone_code failed:%v", err)
			return true, nil
		}

		return true, nil
	}
	return false, errors.New("code not match")
}

func (s *server) VerifyPhoneCode(ctx context.Context, in *verify.PhoneRequest) (*verify.VerifyReply, error) {
	flag, err := checkPhoneCode(in.Phone, in.Code)
	if err != nil {
		return &verify.VerifyReply{Result: false}, err
	}

	return &verify.VerifyReply{Result: flag}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	verify.RegisterVerifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
