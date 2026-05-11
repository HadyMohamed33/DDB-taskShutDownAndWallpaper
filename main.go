package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	fmt.Println("Agent running on port 8081...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	cmd := strings.TrimSpace(string(buf[:n]))
	fmt.Println("Received:", cmd)

	// التعامل مع أوامر الملفات
	if strings.HasPrefix(cmd, "FILE_SEND|") {
		handleFileReceive(conn, cmd)
		return
	}
	if strings.HasPrefix(cmd, "FILE_GET|") {
		handleFileSend(conn, cmd)
		return
	}

	// الأوامر العادية
	res := execute(cmd)
	conn.Write([]byte(res))
}

func handleFileReceive(conn net.Conn, cmd string) {
	parts := strings.Split(cmd, "|")
	if len(parts) != 3 {
		conn.Write([]byte("ERROR"))
		return
	}
	remotePath := parts[1]
	size, _ := strconv.ParseInt(parts[2], 10, 64)

	// إعلام الـ Controller بأنه جاهز
	conn.Write([]byte("READY"))

	// إنشاء الملف
	file, err := os.Create(remotePath)
	if err != nil {
		conn.Write([]byte("ERROR creating file"))
		return
	}
	defer file.Close()

	// استقبال الملف
	var received int64
	buf := make([]byte, 4096)
	for received < size {
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			return
		}
		file.Write(buf[:n])
		received += int64(n)
	}
	fmt.Println("File received:", remotePath, size, "bytes")
}

func handleFileSend(conn net.Conn, cmd string) {
	parts := strings.Split(cmd, "|")
	if len(parts) != 2 {
		conn.Write([]byte("ERROR Invalid command"))
		return
	}
	filePath := parts[1]
	file, err := os.Open(filePath)
	if err != nil {
		conn.Write([]byte(fmt.Sprintf("ERROR %v", err)))
		return
	}
	defer file.Close()
	info, _ := file.Stat()
	// إرسال حجم الملف
	conn.Write([]byte(fmt.Sprintf("OK|%d", info.Size())))
	// إرسال الملف
	io.Copy(conn, file)
	fmt.Println("File sent:", filePath, info.Size(), "bytes")
}

func execute(cmd string) string {
	switch cmd {
	case "lock":
		_ = exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
		return "Locked"
	case "shutdown":
		_ = exec.Command("shutdown", "/s", "/t", "0").Run()
		return "Shutting down"
	case "wallpaper":
		imagePath := "c:\\wall.jpg"
		ps := `(Add-Type '[DllImport("user32.dll")]public static extern int SystemParametersInfo(int uAction,int uParam,string lpvParam,int fuWinIni);' -Name A -Namespace B -PassThru)::SystemParametersInfo(20,0,"` + imagePath + `",3)`
		err := exec.Command("powershell", "-Command", ps).Run()
		if err != nil {
			return "Failed to change wallpaper"
		}
		return "Wallpaper changed"
	default:
		return "Unknown command"
	}
}
