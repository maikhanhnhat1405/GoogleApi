package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// --- PHẦN TRẠM 1: XÁC THỰC (Giữ nguyên logic cũ) ---

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Lỗi lưu token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Mở link này để lấy mã: \n%v\n", authURL)
	fmt.Print("Dán mã vào đây: ")
	var authCode string
	fmt.Scan(&authCode)
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Lỗi đổi mã: %v", err)
	}
	return tok
}

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// --- PHẦN TRẠM 2 & 3: LẤY DỮ LIỆU VÀ XUẤT CSV ---

func main() {
	ctx := context.Background()
	b, _ := os.ReadFile("credentials.json")
	config, _ := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
	client := getClient(config)
	srv, _ := drive.NewService(ctx, option.WithHTTPClient(client))

	// Truy vấn Folder
	r, err := srv.Files.List().
		Q("mimeType = 'application/vnd.google-apps.folder' and trashed = false").
		Fields("files(id, name, createdTime)").Do()
	if err != nil {
		log.Fatalf("Lỗi Drive: %v", err)
	}

	// Xuất File CSV
	csvFile, _ := os.Create("danh_sach_folder.csv")
	defer csvFile.Close()

	// 1. Ghi mã BOM (Giúp Excel hiển thị đúng tiếng Việt có dấu)
	csvFile.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(csvFile)

	// 2. Dùng dấu chấm phẩy (Giúp Excel tự động chia cột tại VN)
	writer.Comma = ';'

	defer writer.Flush()

	// 3. Ghi dữ liệu
	writer.Write([]string{"Tên Thư Mục", "ID", "Ngày Tạo"})
	for _, file := range r.Files {
		writer.Write([]string{file.Name, file.Id, file.CreatedTime})
	}

	fmt.Println("Xong! File CSV đã sẵn sàng.")
}
