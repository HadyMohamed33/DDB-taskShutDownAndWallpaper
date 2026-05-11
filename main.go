package main

import (
	"fmt"
	"net"
	"os/exec"
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

	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)

	cmd := strings.TrimSpace(string(buf[:n]))
	fmt.Println("Received:", cmd)

	res := execute(cmd)

	conn.Write([]byte(res))
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
		// لازم الصورة تكون موجودة هنا:
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
