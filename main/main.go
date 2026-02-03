package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	odooURL            = "http://insights.dtxco.vn:8069" // URL Odoo
	odooDB             = "dtxco_2024"                    // Tên database Odoo
	odooMasterPassword = "odoo@2024"                     // Mật khẩu master của Odoo
	backupDir          = "backups"                       // Thư mục lưu trữ backup
	clientSecretsFile  = "client_secrets.json"           // File xác thực Google Drive API
	credentialsFile    = "credentials.json"              // File lưu token sau khi xác thực
	maxBackupAge       = 5                               // Số ngày giữ lại các bản backup
	googleFolderName   = "odoo_backup"                   // Tên thư mục trên Google Drive để lưu backup
)

// Thêm biến toàn cục để lưu authentication code
var authCode string
var codeChan = make(chan string)

func main() {
	// 1. Tạo thư mục lưu trữ backup nếu chưa tồn tại
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		err := os.MkdirAll(backupDir, os.ModePerm)
		if err != nil {
			fmt.Println("Không thể tạo thư mục backup:", err)
			return
		}
	}

	// 2. Tạo bản sao lưu
	backupFilePath, err := backupOdooDatabase()
	if err != nil {
		fmt.Println("Không thể sao lưu cơ sở dữ liệu:", err)
		return
	}

	// 3. Upload bản sao lưu lên Google Drive
	err = uploadToGoogleDrive(backupFilePath)
	if err != nil {
		fmt.Println("Không thể tải lên Google Drive:", err)
		return
	}

	// 4. Xóa file backup sau khi tải lên Google Drive
	err = os.Remove(backupFilePath)
	if err != nil {
		fmt.Println("Không thể xóa file backup local:", err)
		return
	}
	fmt.Println("Đã xóa file backup local:", backupFilePath)

	// 5. Dọn dẹp các file backup cũ nếu còn sót lại
	cleanOldBackups()
}

// getClient xử lý xác thực OAuth2 và trả về một HTTP client
func getClient(config *oauth2.Config) *http.Client {
	token, err := tokenFromFile(credentialsFile)
	if err != nil {
		token = getTokenFromWeb(config)
		saveToken(credentialsFile, token)
	}
	return config.Client(context.Background(), token)
}

/* // getTokenFromWeb yêu cầu người dùng cấp quyền và lấy token
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Vui lòng truy cập liên kết sau để cấp quyền:\n%v\n", authURL)

	var authCode string
	fmt.Print("Nhập mã xác thực từ trình duyệt: ")
	fmt.Scan(&authCode)

	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		fmt.Println("Không thể lấy token:", err)
		os.Exit(1)
	}
	return token
}

// tokenFromFile đọc token từ file
func tokenFromFile(file string) (*oauth2.Token, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	err = json.Unmarshal(data, &token)
	return &token, err
}

// saveToken lưu token vào file
func saveToken(file string, token *oauth2.Token) {
	data, err := json.Marshal(token)
	if err != nil {
		fmt.Println("Không thể lưu token:", err)
		return
	}
	err = ioutil.WriteFile(file, data, 0600)
	if err != nil {
		fmt.Println("Không thể ghi token vào file:", err)
	}
	fmt.Println("Đã lưu token vào file:", file)
} */

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	// Khởi tạo server local để nhận redirect
	server := &http.Server{Addr: ":8080"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			authCode = code
			fmt.Fprintf(w, `
                <html>
                    <body style="text-align: center; font-family: Arial, sans-serif; margin-top: 50px;">
                        <h2 style="color: #4CAF50;">✓ Xác thực thành công!</h2>
                        <p>Bạn có thể đóng cửa sổ này và quay lại terminal.</p>
                    </body>
                </html>
            `)
			codeChan <- code
			// Tắt server sau khi nhận được code
			go func() {
				server.Shutdown(context.Background())
			}()
		}
	})

	// Chạy server ở port 8080
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Tạo URL xác thực với redirect về localhost:8080
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\nVui lòng truy cập liên kết sau trong trình duyệt để xác thực:\n\n%v\n\n", authURL)
	fmt.Println("Đang chờ xác thực từ trình duyệt...")

	// Đợi code từ callback
	code := <-codeChan

	// Trao đổi code lấy token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("Không thể lấy token: %v", err)
	}

	return token
}

// tokenFromFile đọc token từ file
func tokenFromFile(file string) (*oauth2.Token, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	err = json.Unmarshal(data, &token)
	return &token, err
}

func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Lưu token vào file: %s\n", file)
	data, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Không thể marshal token: %v", err)
	}
	if err := ioutil.WriteFile(file, data, 0600); err != nil {
		log.Fatalf("Không thể lưu token: %v", err)
	}
}

// getOrCreateFolderID tìm hoặc tạo thư mục trên Google Drive
func getOrCreateFolderID(srv *drive.Service, folderName string) (string, error) {
	// Tìm thư mục với tên folderName trong "My Drive"
	query := fmt.Sprintf("mimeType='application/vnd.google-apps.folder' and trashed=false and name='%s'", folderName)
	res, err := srv.Files.List().Q(query).Fields("files(id, name)").Do()
	if err != nil {
		return "", fmt.Errorf("không thể tìm thư mục: %v", err)
	}

	if len(res.Files) > 0 {
		return res.Files[0].Id, nil
	}

	// Tạo thư mục nếu không tìm thấy
	folder := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	createdFolder, err := srv.Files.Create(folder).Fields("id").Do()
	if err != nil {
		return "", fmt.Errorf("không thể tạo thư mục: %v", err)
	}
	return createdFolder.Id, nil
}

// uploadToGoogleDrive tải file lên Google Drive
func uploadToGoogleDrive(filePath string) error {
	fmt.Println("Đang tải file lên Google Drive:", filePath)

	// Đọc file client_secrets.json
	data, err := ioutil.ReadFile(clientSecretsFile)
	if err != nil {
		return fmt.Errorf("Không thể đọc file client_secrets.json: %v", err)
	}

	// Tạo cấu hình OAuth2
	config, err := google.ConfigFromJSON(data, drive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("Không thể tạo cấu hình OAuth2: %v", err)
	}

	// Xác thực và tạo service Google Drive
	client := getClient(config)
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("Không thể tạo service Google Drive: %v", err)
	}

	// Tạo hoặc tìm thư mục "odoo_backup" trong "My Drive"
	backupFolderID, err := getOrCreateFolderID(srv, googleFolderName)
	if err != nil {
		return fmt.Errorf("Không thể tìm hoặc tạo thư mục odoo_backup: %v", err)
	}

	// Tải file lên thư mục "odoo_backup"
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	driveFile := &drive.File{
		Name:    filepath.Base(filePath),
		Parents: []string{backupFolderID},
	}
	_, err = srv.Files.Create(driveFile).Media(file).Do()
	if err != nil {
		return fmt.Errorf("Lỗi tải lên Google Drive: %v", err)
	}

	fmt.Println("Đã tải lên thành công!")

	// Gọi hàm xóa các file backup cũ trên Google Drive
	err = deleteOldBackupsOnDrive(srv, backupFolderID)
	if err != nil {
		fmt.Println("Không thể xóa file cũ trên Google Drive:", err)
	}

	return nil
}

// backupOdooDatabase tạo bản sao lưu cơ sở dữ liệu Odoo và lưu vào thư mục backup
func backupOdooDatabase() (string, error) {
	fmt.Println("Đang sao lưu cơ sở dữ liệu...")

	// Gửi request tới Odoo để tạo backup
	url := fmt.Sprintf("%s/web/database/backup", odooURL)
	payload := fmt.Sprintf("master_pwd=%s&name=%s&backup_format=zip", odooMasterPassword, odooDB)

	resp, err := http.Post(url, "application/x-www-form-urlencoded", bytes.NewBufferString(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("Lỗi từ Odoo: %s", string(body))
	}

	// Lưu file backup vào thư mục
	backupFileName := fmt.Sprintf("%s_%s.zip", odooDB, time.Now().Format("20060102_150405"))
	backupFilePath := filepath.Join(backupDir, backupFileName)

	file, err := os.Create(backupFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("Đã sao lưu thành công:", backupFilePath)
	return backupFilePath, nil
}

// cleanOldBackups xóa các file backup cũ hơn maxBackupAge ngày
func cleanOldBackups() {
	fmt.Println("Đang xóa các file backup cũ hơn", maxBackupAge, "ngày...")

	files, err := os.ReadDir(backupDir)
	if err != nil {
		fmt.Println("Không thể đọc thư mục backup:", err)
		return
	}

	now := time.Now()
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(backupDir, file.Name())
			info, err := os.Stat(filePath)
			if err != nil {
				fmt.Println("Không thể đọc thông tin file:", err)
				continue
			}

			if now.Sub(info.ModTime()).Hours() > float64(maxBackupAge*24) {
				err := os.Remove(filePath)
				if err != nil {
					fmt.Println("Không thể xóa file:", filePath, err)
					continue
				}
				fmt.Println("Đã xóa file:", filePath)
			}
		}
	}
}

// deleteOldBackupsOnDrive xóa các file backup cũ hơn maxBackupAge ngày trên Google Drive
func deleteOldBackupsOnDrive(srv *drive.Service, backupFolderID string) error {
	fmt.Println("Đang kiểm tra và xóa các file backup cũ trên Google Drive...")

	// Truy vấn các file trong thư mục "odoo_backup"
	query := fmt.Sprintf("'%s' in parents and mimeType!='application/vnd.google-apps.folder' and trashed=false", backupFolderID)
	files, err := srv.Files.List().Q(query).Fields("files(id, name, createdTime)").Do()
	if err != nil {
		return fmt.Errorf("Không thể lấy danh sách file từ Google Drive: %v", err)
	}

	if len(files.Files) == 0 {
		fmt.Println("Không tìm thấy file backup nào để xóa trên Google Drive.")
		return nil
	}

	now := time.Now()
	for _, file := range files.Files {
		// Chuyển đổi thời gian từ Google Drive
		fileTime, err := time.Parse(time.RFC3339, file.CreatedTime)
		if err != nil {
			fmt.Printf("Lỗi khi đọc thời gian tạo file %s: %v\n", file.Name, err)
			continue
		}

		// Xóa file nếu nó đã quá maxBackupAge ngày
		if now.Sub(fileTime).Hours() > float64(maxBackupAge*24) {
			fmt.Printf("Đang xóa file: %s\n", file.Name)
			err := srv.Files.Delete(file.Id).Do()
			if err != nil {
				fmt.Printf("Không thể xóa file %s trên Google Drive: %v\n", file.Name, err)
			} else {
				fmt.Printf("Đã xóa file: %s\n", file.Name)
			}
		}
	}

	fmt.Println("Hoàn tất quá trình kiểm tra và xóa file cũ trên Google Drive.")
	return nil
}
