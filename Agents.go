package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

var agents = map[string]string{
	"pc1": "172.21.16.1:8081",
	"pc2": "192.168.1.16:8081",
	"pc3": "192.168.137.17:8081",
	"pc4": "192.168.137.160:8081",
	"pc5": "192.168.137.149:8081",
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n==== Controller ====")
		fmt.Println("1) Add Agent")
		fmt.Println("2) Show Agents")
		fmt.Println("3) Send Command")
		fmt.Println("4) Send File to Agent")
		fmt.Println("5) Get File from Agent")
		fmt.Println("6) Exit")
		fmt.Print("Choose: ")

		ch, _ := reader.ReadString('\n')
		ch = strings.TrimSpace(ch)

		switch ch {
		case "1":
			add(reader)
		case "2":
			show()
		case "3":
			sendMenu(reader)
		case "4":
			sendFileMenu(reader)
		case "5":
			getFileMenu(reader)
		case "6":
			return
		}
	}
}

func add(r *bufio.Reader) {
	fmt.Print("Name: ")
	name, _ := r.ReadString('\n')
	fmt.Print("IP:PORT: ")
	ip, _ := r.ReadString('\n')
	name = strings.TrimSpace(name)
	ip = strings.TrimSpace(ip)
	agents[name] = ip
	fmt.Println("Added ✔")
}

func show() {
	for k, v := range agents {
		fmt.Println(k, "=>", v)
	}
}

func sendMenu(r *bufio.Reader) {
	fmt.Println("Agents:")
	for k := range agents {
		fmt.Println("-", k)
	}
	fmt.Print("Target (pc1,pc2,.. or all): ")
	target, _ := r.ReadString('\n')
	target = strings.TrimSpace(target)
	fmt.Print("Command (lock/shutdown/wallpaper): ")
	cmd, _ := r.ReadString('\n')
	cmd = strings.TrimSpace(cmd)

	if target == "all" {
		for _, ip := range agents {
			go sendCommand(ip, cmd)
		}
		return
	}
	names := strings.Split(target, ",")
	for _, n := range names {
		n = strings.TrimSpace(n)
		if ip, ok := agents[n]; ok {
			go sendCommand(ip, cmd)
		}
	}
}

func sendCommand(addr, cmd string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println(addr, "❌", err)
		return
	}
	defer conn.Close()
	conn.Write([]byte(cmd))
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	fmt.Println(addr, "➡", string(buf[:n]))
}

// ===================== إرسال ملف إلى الـ Agent =====================
func sendFileMenu(r *bufio.Reader) {
	fmt.Println("Agents:")
	for k := range agents {
		fmt.Println("-", k)
	}
	fmt.Print("Target (pc1,pc2,.. or all): ")
	target, _ := r.ReadString('\n')
	target = strings.TrimSpace(target)
	fmt.Print("Local file path to send: ")
	localPath, _ := r.ReadString('\n')
	localPath = strings.TrimSpace(localPath)
	fmt.Print("Remote destination path (on agent): ")
	remotePath, _ := r.ReadString('\n')
	remotePath = strings.TrimSpace(remotePath)

	if target == "all" {
		for _, ip := range agents {
			go sendFile(ip, localPath, remotePath)
		}
		return
	}
	names := strings.Split(target, ",")
	for _, n := range names {
		n = strings.TrimSpace(n)
		if ip, ok := agents[n]; ok {
			go sendFile(ip, localPath, remotePath)
		}
	}
}

func sendFile(addr, localPath, remotePath string) {
	file, err := os.Open(localPath)
	if err != nil {
		fmt.Println(addr, "❌ Cannot open local file:", err)
		return
	}
	defer file.Close()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println(addr, "❌", err)
		return
	}
	defer conn.Close()

	// بروتوكول بسيط: FILE_SEND|remotePath|size
	info, _ := file.Stat()
	cmd := fmt.Sprintf("FILE_SEND|%s|%d", remotePath, info.Size())
	conn.Write([]byte(cmd))

	// انتظر تأكيد
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	if string(buf[:n]) != "READY" {
		fmt.Println(addr, "❌ Agent not ready")
		return
	}

	// أرسل الملف
	written, err := io.Copy(conn, file)
	if err != nil {
		fmt.Println(addr, "❌ File send error:", err)
		return
	}
	fmt.Println(addr, "✅ File sent:", written, "bytes to", remotePath)
}

// ===================== استلام ملف من الـ Agent =====================
func getFileMenu(r *bufio.Reader) {
	fmt.Println("Agents:")
	for k := range agents {
		fmt.Println("-", k)
	}
	fmt.Print("Target (pc1,pc2,.. or all): ")
	target, _ := r.ReadString('\n')
	target = strings.TrimSpace(target)
	fmt.Print("Remote file path on agent: ")
	remotePath, _ := r.ReadString('\n')
	remotePath = strings.TrimSpace(remotePath)
	fmt.Print("Local save path: ")
	localPath, _ := r.ReadString('\n')
	localPath = strings.TrimSpace(localPath)

	if target == "all" {
		for _, ip := range agents {
			go getFile(ip, remotePath, localPath)
		}
		return
	}
	names := strings.Split(target, ",")
	for _, n := range names {
		n = strings.TrimSpace(n)
		if ip, ok := agents[n]; ok {
			go getFile(ip, remotePath, localPath)
		}
	}
}

func getFile(addr, remotePath, localPath string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println(addr, "❌", err)
		return
	}
	defer conn.Close()

	cmd := fmt.Sprintf("FILE_GET|%s", remotePath)
	conn.Write([]byte(cmd))

	// اقرأ الرد: OK|size أو ERROR
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println(addr, "❌ Read error:", err)
		return
	}
	response := string(buf[:n])
	if strings.HasPrefix(response, "ERROR") {
		fmt.Println(addr, "❌", response)
		return
	}
	if !strings.HasPrefix(response, "OK|") {
		fmt.Println(addr, "❌ Invalid response")
		return
	}

	// استخراج حجم الملف
	var size int64
	fmt.Sscanf(response, "OK|%d", &size)

	// إنشاء الملف المحلي
	outFile, err := os.Create(localPath)
	if err != nil {
		fmt.Println(addr, "❌ Cannot create local file:", err)
		return
	}
	defer outFile.Close()

	// قراءة باقي البيانات (الملف)
	var received int64
	for received < size {
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println(addr, "❌ Receive error:", err)
			return
		}
		outFile.Write(buf[:n])
		received += int64(n)
	}
	fmt.Println(addr, "✅ File received:", received, "bytes saved to", localPath)
}
