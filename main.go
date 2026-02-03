package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"context"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Cấu hình
const (
	ODOO_URL                = "http://localhost:8069"
	ODOO_DB                 = "your_database_name"
	ODOO_USERNAME           = "your_admin_username"
	ODOO_PASSWORD           = "your_admin_password"
	BACKUP_DIR              = "backups"
	GOOGLE_CREDENTIALS_FILE = "client_secrets.json"
)

// authenticateGoogleDrive xác thực với Google Drive
func authenticateGoogleDrive() (*drive.Service, error) {
	// Đọc thông tin xác thực từ tệp JSON (credentials.json)
	credentials, err := os.ReadFile(GOOGLE_CREDENTIALS_FILE)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc tệp credentials: %v", err)
	}

	// Tạo cấu hình OAuth2 từ credentials
	config, err := google.ConfigFromJSON(credentials, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("không thể tạo config từ credentials: %v", err)
	}

	// Lấy mã token từ người dùng hoặc lưu token để sử dụng lại
	token, err := getTokenFromWeb(config)
	if err != nil {
		return nil, fmt.Errorf("không thể lấy token: %v", err)
	}

	// Tạo dịch vụ Google Drive
	client := config.Client(context.Background(), token)
	service, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("không thể tạo service Google Drive: %v", err)
	}

	return service, nil
}

// getTokenFromWeb thực hiện quy trình xác thực để lấy token từ người dùng
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Mở liên kết sau và nhập mã xác thực: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("không thể đọc mã xác thực: %v", err)
	}

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("không thể trao đổi mã xác thực: %v", err)
	}

	return token, nil
}

// backupOdooDatabase kết nối tới Odoo và sao lưu CSDL
func backupOdooDatabase() (string, error) {
	fmt.Println("Bắt đầu sao lưu cơ sở dữ liệu...")

	// Tạo request
	odoo_url := fmt.Sprintf("%s/web/database/backup", ODOO_URL)
	params := url.Values{
		"master_pwd":    {"odoo@2024"},
		"name":          {"dtxco_2024"},
		"backup_format": {"zip"},
	}

	// Gửi POST request
	resp, err := http.PostForm(odoo_url, params)
	if err != nil {
		return "", fmt.Errorf("lỗi khi gửi request backup: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("không thể backup database, status code: %d", resp.StatusCode)
	}

	// Tạo tên file backup
	backupFilename := fmt.Sprintf("%s_%s.zip", ODOO_DB, time.Now().Format("20060102_150405"))
	backupPath := filepath.Join(BACKUP_DIR, backupFilename)

	// Tạo file
	out, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("không thể tạo file backup: %v", err)
	}
	defer out.Close()

	// Copy dữ liệu
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("không thể ghi file backup: %v", err)
	}

	fmt.Printf("Đã tạo bản sao lưu: %s\n", backupPath)
	return backupPath, nil
}

// uploadToGoogleDrive tải file lên Google Drive
func uploadToGoogleDrive(srv *drive.Service, filePath string) error {
	fmt.Printf("Đang tải lên Google Drive: %s\n", filePath)

	file := &drive.File{
		Name: filepath.Base(filePath),
	}

	// Mở file để đọc
	content, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("không thể mở file: %v", err)
	}
	defer content.Close()

	// Upload file
	_, err = srv.Files.Create(file).Media(content).Do()
	if err != nil {
		return fmt.Errorf("không thể upload file: %v", err)
	}

	fmt.Printf("Đã tải lên thành công: %s\n", filepath.Base(filePath))
	return nil
}

// cleanOldBackups xóa các file backup cũ hơn 5 ngày
func cleanOldBackups() error {
	fmt.Println("Đang dọn dẹp các file backup cũ...")

	files, err := os.ReadDir(BACKUP_DIR)
	if err != nil {
		return fmt.Errorf("không thể đọc thư mục backup: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(BACKUP_DIR, file.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > 5*24*time.Hour {
			err := os.Remove(filePath)
			if err != nil {
				fmt.Printf("Không thể xóa file %s: %v\n", filePath, err)
				continue
			}
			fmt.Printf("Đã xóa file backup cũ: %s\n", filePath)
		}
	}
	return nil
}

func main2() {
	// Tạo thư mục backup nếu chưa tồn tại
	if err := os.MkdirAll(BACKUP_DIR, 0755); err != nil {
		fmt.Printf("Không thể tạo thư mục backup: %v\n", err)
		return
	}

	// Xác thực Google Drive
	srv, err := authenticateGoogleDrive()
	if err != nil {
		fmt.Printf("Lỗi xác thực Google Drive: %v\n", err)
		return
	}

	// Sao lưu cơ sở dữ liệu
	backupPath, err := backupOdooDatabase()
	if err != nil {
		fmt.Printf("Lỗi khi backup database: %v\n", err)
		return
	}

	// Tải lên Google Drive
	if err := uploadToGoogleDrive(srv, backupPath); err != nil {
		fmt.Printf("Lỗi khi upload lên Google Drive: %v\n", err)
		return
	}

	// Dọn dẹp file backup cũ
	if err := cleanOldBackups(); err != nil {
		fmt.Printf("Lỗi khi dọn dẹp file backup cũ: %v\n", err)
	}
}
