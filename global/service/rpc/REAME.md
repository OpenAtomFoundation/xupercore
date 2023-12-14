# RPC 接口

## XRandom

- 默认端口：37101

实例代码：
```go
func main() {
	rpcServer := fmt.sprintf("%s:%d", host, port)
	conn, err := grpc.Dial(rpcServer, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("create connection error: %s\n", err)
		return
	}
	
	client := pb.NewRandomClient(conn)
	fmt.Print("client created\n")

	request := pb.QueryRandomNumberRequest{
		// ...
	}

	response, err := client.QueryRandomNumber(context.Background(), &request)
	if err != nil {
		fmt.Printf("call error: %s\n", err)
		return
	}

	fmt.Printf("call response: %+v", *response)
	return
}
```