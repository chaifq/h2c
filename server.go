package main

import (
    "crypto/tls"
    "fmt"
    "io/ioutil"
    "net"
    "net/http"
    "strings"
)

func main() {
    // 创建一个TCP监听器，监听8080端口
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer ln.Close()

    // 创建一个HTTP服务器，使用自定义的处理程序
    server := &http.Server{
        Handler: http.HandlerFunc(handleRequest),
        TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
    }

    // 开始监听TCP连接并在每个连接上处理请求
    for {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Println(err)
            continue
        }

        // 检查连接是否是H2C连接
        if isH2C(conn) {
            // 如果是H2C连接，则调用处理程序并启动HTTP/2流
            go func() {
                if err := server.ServeConn(conn); err != nil {
                    fmt.Println(err)
                }
            }()
        } else {
            // 如果不是H2C连接，则关闭连接
            conn.Close()
        }
    }
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 处理HTTP请求并向客户端发送响应
    w.Write([]byte("Hello, World!\n"))
}

func isH2C(conn net.Conn) bool {
    // 从连接中读取前24个字节
    buf := make([]byte, 24)
    n, err := conn.Read(buf)
    if err != nil || n < 24 {
        return false
    }

    // 检查前12个字节是否与预期的H2C前缀匹配
    prefix := []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
    if !bytes.Equal(buf[:12], prefix) {
        return false
    }

    // 检查后12个字节是否表示一个有效的HTTP/1.1请求
    method, target, version, err := http.ParseRequestLine(string(buf[12:]))
    if err != nil || !strings.HasPrefix(version, "HTTP/1.") {
        return false
    }

    // 检查请求头是否包含必要的协议升级头
    upgrade := r.Header.Get("Upgrade")
    connection := r.Header.Get("Connection")
    if upgrade != "h2c" || !strings.Contains(strings.ToLower(connection), "upgrade") {
        return false
    }

    return true
}
